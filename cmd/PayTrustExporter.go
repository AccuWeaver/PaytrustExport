package main

import (
	"PayTrust/onepassword"
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/playwright-community/playwright-go"
)

var lvl = new(slog.LevelVar)
var logger *slog.Logger

func main() {

	// Arguments for 1Password vault and tags to use to get password
	vault := flag.String("vault", "Personal", "1Password vault")
	tags := flag.String("tags", "Paytrust", "1Password tags")

	// Password to use if not using 1Password
	password := flag.String("1password_pass", "", "Password to use if not using 1Password")

	// Paytrust login URL argument
	url := flag.String("url", "https://login.billscenter.paytrust.com/3004/", "Paytrust login URL")

	// Paytrust login username argument
	username := flag.String("username", "", "Paytrust login username")

	// Debug flag
	debug := flag.Bool("debug", false, "Debug flag")

	// Parse the flags
	flag.Parse()

	lvl.Set(slog.LevelInfo)

	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
	}))

	// if debug, set the log level to debug
	if *debug {
		lvl.Set(slog.LevelDebug)
	}

	// If no password, get it from 1Password
	if *password == "" {
		logger.Debug("Getting password from 1Password")
		vaultUserName, vaultPassword := GetPasswordFromVault(vault, tags)
		username = &vaultUserName
		password = &vaultPassword

	}

	// Set up our run options
	runOption := &playwright.RunOptions{
		SkipInstallBrowsers: true,
	}
	// Install playwright
	err := playwright.Install(runOption)
	if err != nil {
		log.Fatalf("could not install playwright dependencies: %v", err)
	}

	// Playwright variable
	var pw *playwright.Playwright
	// Start playwright running
	pw, err = playwright.Run()
	if err != nil {
		log.Fatalf("could not start playwright: %v", err)
	}

	// Browser to use
	var browser playwright.Browser
	browser, err = pw.Chromium.Launch(
		playwright.BrowserTypeLaunchOptions{
			// Apparently defaults to true
			Headless: playwright.Bool(false),
			Channel:  playwright.String("chrome"),
		},
	)
	if err != nil {
		log.Fatalf("could not launch browser: %v", err)
	}

	// Page object
	var page playwright.Page
	page, err = browser.NewPage()
	if err != nil {
		log.Fatalf("could not create page: %v", err)
	}

	// Open Paytrust login page ...
	var response playwright.Response
	if response, err = page.Goto(*url); err != nil {
		log.Fatalf("could not go to %v: %v", *url, err)
	}
	if response.Status() != 200 {
		log.Fatalf("could not goto: %v", response.Status())
	}

	// Enter username
	userName := page.Locator("#UserName input[type=text]")

	// Get the count to see if we found it ...
	var UserNameInputCount int

	// See if we found it ...
	UserNameInputCount, err = userName.Count()
	if err != nil {
		log.Fatalf("could not get userName: %v", err)
	}
	// If we didn't find it, we are done
	if UserNameInputCount == 0 {
		log.Fatalf("could not find userName")
	}

	// Fill in the username
	userName.Fill(*username)
	//log.Printf("Element: %#v", userName)
	logger.Debug("UserName filled in", *username)

	// Click continue
	continueButton := page.Locator("#UserName > div.buttons > button")
	var continueButtonCount int
	continueButtonCount, err = continueButton.Count()
	if err != nil {
		log.Fatalf("could not get continueButton: %v", err)
	}
	if continueButtonCount == 0 {
		log.Fatalf("could not find continueButton")
	}
	//log.Printf("continuButton: %#v", continueButton)
	continueButton.Click()
	logger.Debug("Continue buttong clicked")

	// Bring it back to the front
	page.BringToFront()

	// Get the form
	form := page.Locator("div.page.authentication > div.region.right > form")
	formHTML, err := form.Evaluate("el => el.outerHTML", nil)
	formCount, err := page.Locator("div.page.authentication > div.region.right > form").Count()
	if err != nil {
		logger.Error("Could not get form: %v", err)
	}
	// This is where the phone call could happen ...
	// https://login.billscenter.paytrust.com/3004/OOBA/Preview
	if formCount == 1 && strings.Contains(fmt.Sprintf("%v", formHTML), "OOBA/Preview") {
		logger.Info("Handling the logic to for phone verification")
		// We need to wait for the phone call to finish
		ManualStepCompletion("Phone call received")
	}

	// Enter password
	userName = page.Locator("#Password")
	//log.Printf("Element: %#v", userName)
	var passwordInputCount int
	passwordInputCount, err = userName.Count()
	if err != nil {
		log.Fatalf("could not get userName: %v", err)
	}
	if passwordInputCount == 0 {
		log.Fatalf("could not find userName")
	}
	userName.Fill(*password)
	logger.Debug("Password filled in", *password)

	// Click continue
	// Selector from browser: body > div > div.region.right > form:nth-child(3) > div.buttons > button.button.primary
	signonButton := page.Locator("div.buttons > button.button.primary")
	signonButtonCount, err := signonButton.Count()
	if err != nil {
		log.Fatalf("could not get signonButton: %v", err)
	}
	if signonButtonCount == 0 {
		log.Fatalf("could not find signonButton")
	}
	//log.Printf("signonButton: %#v", signonButton)
	signonButton.Click()
	logger.Debug("Signon clicked")

	// Should be logged in now
	// 	// body > div.ui-dialog.ui-corner-all.ui-widget.ui-widget-content.ui-front.EPP > div.ui-dialog-titlebar.ui-corner-all.ui-widget-header.ui-helper-clearfix > button
	noticeClear := page.Locator(`button[title="Close"]`)
	log.Printf("noticeClear: %#v", noticeClear)
	noticeClearCount, err := noticeClear.Count()
	if err != nil {
		log.Fatalf("could not get noticeClear dropDownItemsCount: %v", err)
	}
	// If there is a notice, clear it
	if noticeClearCount == 1 {
		logger.Debug("Clearing notice")
		noticeClear.Click()
	}

	// Click the Reports tab
	tabSelect := page.Locator("#RightRegionContentPlaceHolder_SidebarTabMenu_PaymentHistory > a")
	//log.Printf("tabSelect: %#v", tabSelect)
	tabSelectCount, err := tabSelect.Count()
	if err != nil {
		log.Fatalf("could not get tabSelect dropDownItemsCount: %v", err)
	}
	//log.Printf("tabSelectCount: %#v", tabSelectCount)
	if tabSelectCount == 0 {
		log.Fatalf("could not find tabSelect")
	}
	tabSelect.Click()
	logger.Debug("Reports tab clicked")

	// Click the Spending Report link
	spendingReportLink := page.Locator("#RightRegionContentPlaceHolder_PaymentHistory_PaymentTable_ReportsLink_Input")
	//log.Printf("spendingReportLink: %#v", spendingReportLink)
	spendingReportLinkCount, err := spendingReportLink.Count()
	if err != nil {
		log.Fatalf("could not get spendingReportLink dropDownItemsCount: %v", err)
	}
	if spendingReportLinkCount == 0 {
		log.Fatalf("could not find spendingReportLink")
	}
	spendingReportLink.Click()
	logger.Debug("Spending Report link clicked")

	// Get the view options
	reportOptions := page.Locator("#Reports_ViewReportsReportDropdown_Input_Responsive")
	//log.Printf("reportOptions: %#v", reportOptions)
	selectionItemCount, err := reportOptions.Count()
	if err != nil {
		log.Fatalf("could not get reportOptions dropDownItemsCount: %v", err)
	}
	if selectionItemCount == 0 {
		log.Fatalf("could not find reportOptions")
	}
	reportOptions.Click()
	logger.Debug("Report options clicked")

	// Get the drop down items
	dropDownItems := page.Locator("#Reports_ViewReportsReportDropdown_Input_Responsive > div.responsiveDropDownOptions > div > div")
	dropDownItemsCount, err := dropDownItems.Count()
	if err != nil {
		log.Fatalf("could not get dropDownItemsCount: %v", err)
	}
	if dropDownItemsCount == 0 {
		log.Fatalf("could not find dropDownItems")
	}
	logger.Debug("dropDownItems: %#v", dropDownItemsCount)

	// Get the drop down items
	billButtons, err := dropDownItems.All()
	if err != nil {
		log.Fatalf("could not get billButtons: %v", err)
	}
	logger.Debug("billButtons: %#v", billButtons)

	var foundReport bool
	// All dates value
	allDates := "Include All Dates"
	for _, item := range billButtons {
		text, err := item.TextContent()
		if err != nil {
			log.Fatalf("could not get text: %v", err)
		}
		if strings.Contains(text, allDates) {
			log.Printf("Found All Dates")
			foundReport = true
			break
		}
	}
	if !foundReport {
		log.Fatalf("could not find report for %#v", allDates)
	}
	logger.Debug("Found All Dates in selections")

	// So now we have the report up, get all the PDF links
	PDFLinks := page.Locator("#subcategory\\ Fis\\.Epp\\.DomainModel\\.BillPay\\.ReportGroup > tbody > tr > td.column.bill > billButton")
	dropDownItemsCount, err = PDFLinks.Count()
	if err != nil {
		log.Fatalf("could not get dropDownItemsCount of links: %v", err)
	}
	log.Printf("PDFLinks: %#v", dropDownItemsCount)
	// No PDF links, so we are done
	if dropDownItemsCount == 0 {
		log.Fatalf("No PDFLinks")
	}
	billButtons, err = PDFLinks.All()
	if err != nil {
		log.Fatalf("could not get billButtons: %v", err)
	}
	if len(billButtons) == 0 {
		log.Fatalf("No billButtons")
	}
	logger.Debug("billButtons: %#v", billButtons)

	// Loop through all the links ....
	for _, billButton := range billButtons {
		var html interface{}
		html, err = billButton.Evaluate("el => el.outerHTML", nil)
		if err != nil {
			log.Fatalf("could not get html: %v", err)
		}
		logger.Debug("html: %#v", html)

		// Open the window ...
		billButton.Click()
		logger.Debug("billButton clicked")

		// Get image billButton
		billWindowButton := page.Locator("#ViewBills > div.view.extraLarge > div > div.container.sectionsContainer > div.section.content.contentSection.clear > div.area.billimage.clear > div.areaHeader > span.newWindow > a")
		billWindowButtonCount, err := billWindowButton.Count()
		if err != nil {
			log.Printf("could not get image billButton dropDownItemsCount: %v", err)
			continue
		}
		// Has an image billButton in the window, so click i
		if billWindowButtonCount == 1 {
			logger.Debug("Processing billWindowButton")
			// Open the bill window
			billWindowButton.Click()
			logger.Debug("billWindowButton clicked")

			// See if there is an e-bill billButton in the window
			ebillLink := page.Locator("#PaymentSection > div.subcontent > div.sidebar.billButtons > billButton.bill")
			ebillLinkCount, err := ebillLink.Count()
			if err != nil {
				log.Printf("could not get ebill billButton dropDownItemsCount: %v", err)
				continue
			}
			if ebillLinkCount == 1 {
				// We have a bill to download.
				ebillLink.Click()
				logger.Debug("ebillLink clicked")

				pdf := page.Locator("body > embed")
				pdfCount, err := pdf.Count()
				if err != nil {
					log.Fatalf("could not get pdf dropDownItemsCount: %v", err)
				}
				logger.Debug("pdfCount: %#v", pdfCount)

				// Get the OuterHTML of the PDF
				pdfHTML, err := pdf.Evaluate("el => el.outerHTML", nil)
				if err != nil {
					log.Fatalf("could not get html: %v", err)
				}
				logger.Debug("pdfHTML: %#v", pdfHTML)
				// This is where we would download the PDF
				logger.Info("Download PDF for %v", html)

				// Close the PDF window
				closeButton := page.Locator("body > div:nth-child(11) > div.ui-dialog-titlebar.ui-corner-all.ui-widget-header.ui-helper-clearfix > billButton")
				closeButtonCount, err := closeButton.Count()
				if err != nil {
					log.Fatalf("could not get closeButton dropDownItemsCount: %v", err)
					continue
				}
				if closeButtonCount == 1 {
					closeButton.Click()
					logger.Debug("closeButton clicked")
				}
			}
			// Close the image window
			closeButton := page.Locator("body > div:nth-child(11) > div.ui-dialog-titlebar.ui-corner-all.ui-widget-header.ui-helper-clearfix > billButton")
			closeButton.Click()
		}
		// Close the billWindowButton window
		closeBillWindowButton := page.Locator("body > div:nth-child(11) > div.ui-dialog-titlebar.ui-corner-all.ui-widget-header.ui-helper-clearfix > billButton")
		closeBillWindowButtonCount, err := closeBillWindowButton.Count()
		if err != nil {
			log.Fatalf("could not get closeBillWindowButton dropDownItemsCount: %v", err)
		}
		if closeBillWindowButtonCount == 1 {
			closeBillWindowButton.Click()
			logger.Debug("closeBillWindowButton clicked")
		}
	}
}

func GetPasswordFromVault(vault *string, tags *string) (vaultUserName string, vaultPassword string) {
	vaultEntries := onepassword.GetVaultEntries(*vault, *tags)
	log.Printf("vaultEntries: %d", len(vaultEntries))
	passwordCommand := exec.Command("sh", "-c", fmt.Sprintf(`op read op://%v/%v/password`, *vault, vaultEntries[0].ID))
	var passwordStdout, passwordStdErr bytes.Buffer
	passwordCommand.Stdout = &passwordStdout
	passwordCommand.Stderr = &passwordStdErr
	err := passwordCommand.Run()

	if err != nil {
		log.Printf("Error getting password for %v from 1Password: %v", vaultEntries[0].ID, err)
		log.Fatalf("Stderr: %v", string(passwordStdErr.Bytes()))
	}

	vaultPassword = strings.TrimSpace(string(passwordStdout.Bytes()))
	logger.Debug("vaultPassword: %v", vaultPassword)

	userNameCommand := exec.Command("sh", "-c", fmt.Sprintf(`op read op://%v/%v/username`, *vault, vaultEntries[0].ID))
	var userNameStdOut, userNameStdErr bytes.Buffer
	userNameCommand.Stdout = &userNameStdOut
	userNameCommand.Stderr = &userNameStdErr
	err = userNameCommand.Run()

	if err != nil {
		log.Printf("Error getting username for %v from 1Password: %v", vaultEntries[0].ID, err)
		log.Fatalf("Stderr: %v", string(userNameStdErr.Bytes()))
	}
	vaultUserName = strings.TrimSpace(string(userNameStdOut.Bytes()))
	logger.Debug("vaultUserName: %v", vaultUserName)
	return vaultUserName, vaultPassword
}

func ManualStepCompletion(taskString string) {
	// for forced input
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("\n%v? ", taskString)
		text, _ := reader.ReadString('\n')
		// convert CRLF to LF
		text = strings.Replace(text, "\n", "", -1)

		if strings.Compare("yes", text) == 0 {
			break
		}

	}
}
