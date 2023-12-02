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
	"regexp"
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
	err = userName.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(5000), // wait for 5 seconds
	})
	if err != nil {
		log.Fatalf("could not wait for userName: %v", err)
	}

	// Fill in the username
	err = userName.Fill(*username)
	if err != nil {
		log.Fatalf("could not fill in userName: %v", err)
	}
	//log.Printf("Element: %#v", userName)
	logger.Debug("UserName filled in", *username)

	// Click continue
	continueButton := page.Locator("#UserName > div.buttons > button")
	err = continueButton.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(5000), // wait for 5 seconds
	})
	if err != nil {
		log.Fatalf("could not wait for continue button: %v", err)
	}

	//log.Printf("continuButton: %#v", continueButton)
	err = continueButton.Click()
	if err != nil {
		log.Fatalf("could not click continueButton: %v", err)
	}
	logger.Debug("Continue button clicked")

	// Get the form
	form := page.Locator("div.page.authentication > div.region.right > form")
	err = form.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(5000), // wait for 5 seconds
	})
	if err != nil {
		log.Fatalf("could not wait for form: %v", err)
	}
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
	err = userName.Fill(*password)
	if err != nil {
		log.Fatalf("could not fill in userName: %v", err)
	}
	logger.Debug("Password filled in", *password)

	// Click continue
	// Selector from browser: body > div > div.region.right > form:nth-child(3) > div.buttons > button.button.primary
	signonButton := page.Locator("form:nth-child(3) > div.buttons > button.button.primary")
	signonButtonCount, err := signonButton.Count()
	if err != nil {
		log.Fatalf("could not get signonButton: %v", err)
	}
	if signonButtonCount == 0 {
		log.Fatalf("could not find signonButton")
	}
	//log.Printf("signonButton: %#v", signonButton)
	err = signonButton.Click()
	if err != nil {
		log.Fatalf("could not click signonButton: %v", err)
	}
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
	err = tabSelect.Click()
	if err != nil {
		log.Fatalf("could not click tabSelect: %v", err)
	}
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
	err = spendingReportLink.Click()
	if err != nil {
		log.Fatalf("could not click spendingReportLink: %v", err)
	}
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
	reportSelectItems, err := dropDownItems.All()
	if err != nil {
		log.Fatalf("could not get reportSelectItems: %v", err)
	}
	logger.Debug("reportSelectItems: %#v", reportSelectItems)

	var foundReport bool
	// All dates value
	allDates := "Include All Dates"
	for _, item := range reportSelectItems {
		text, err := item.TextContent()
		if err != nil {
			log.Fatalf("could not get text: %v", err)
		}
		if strings.Contains(text, allDates) {
			log.Printf("Found All Dates")
			foundReport = true
			err = item.Click()
			if err != nil {
				log.Fatalf("could not click All Dates: %v", err)
			}
			break
		}
	}
	if !foundReport {
		log.Fatalf("could not find report for %#v", allDates)
	}
	logger.Debug("Found All Dates in selections")

	// Click the All Dates report

	// So now we have the report up, get all the PDF links
	// #subcategory\ Fis\.Epp\.DomainModel\.BillPay\.ReportGroup > tbody > tr:nth-child(2) > td.column.bill > button
	var PDFLinksCount int
	PDFLinks := page.Locator("button.bill.billIcon")
	PDFLinksCount, err = PDFLinks.Count()
	if err != nil {
		log.Fatalf("could not get dropDownItemsCount of links: %v", err)
	}
	log.Printf("PDFLinks: %#v", PDFLinksCount)
	// No PDF links, so we are done
	if PDFLinksCount == 0 {
		log.Fatalf("No PDFLinks")
	}
	var PDFLinksAll []playwright.Locator
	PDFLinksAll, err = PDFLinks.All()
	if err != nil {
		log.Fatalf("could not get reportSelectItems: %v", err)
	}
	if len(PDFLinksAll) == 0 {
		log.Fatalf("No PDFLinksAll")
	}
	logger.Debug("PDFLinksAll: %#v", PDFLinksAll)

	// Loop through all the links ....
	for _, PDFLink := range PDFLinksAll {
		var html interface{}
		html, err = PDFLink.Evaluate("el => el.outerHTML", nil)
		if err != nil {
			log.Fatalf("could not get html: %v", err)
		}
		logger.Debug("PDFLink html: %#v", html)

		// Open the window ...
		err = PDFLink.Click()
		if err != nil {
			log.Fatalf("could not click PDFLink: %v", err)
		}
		logger.Debug("PDFLink clicked")

		// Get image PDFLink
		billWindowButton := page.Locator("#ViewBills > div.view.extraLarge > div > div.container.sectionsContainer > div.section.content.contentSection.clear > div.area.billimage.clear > div.areaHeader > span.newWindow > a")
		billWindowButtonCount, err := billWindowButton.Count()
		if err != nil {
			log.Printf("could not get image PDFLink dropDownItemsCount: %v", err)
			continue
		}
		// Has an image PDFLink in the window, so click i
		if billWindowButtonCount == 1 {
			var outerHtml interface{}
			outerHtml, err = billWindowButton.Evaluate("el => el.outerHTML", nil)
			if err != nil {
				log.Fatalf("could not get html: %v", err)
			}
			linkText := fmt.Sprintf("%v", outerHtml)

			// <a href="([^"])+
			re := regexp.MustCompile(`<a href="([^"]*?)"`)
			res := re.FindAllStringSubmatch(linkText, 1)
			log.Printf("linkText href: %#v", res[0][1])

			// Close the window
			// body > div:nth-child(12) > div.ui-dialog-titlebar.ui-corner-all.ui-widget-header.ui-helper-clearfix > button
			closeButton := page.Locator("div.section.buttons.buttonSection > button.ui-dialog-titlebar-close")
			closeButtonCount, err := closeButton.Count()
			if err != nil {
				log.Fatalf("could not get closeButton dropDownItemsCount: %v", err)
				continue
			}
			if closeButtonCount != 1 {
				log.Fatalf("could not find closeButton")
			}
			err = closeButton.Click()
			if err != nil {
				log.Fatalf("could not click closeButton: %v", err)
			}
			continue
			// Download the PDF using the URL we found
			// https://login.billscenter.paytrust.com/3004/Document/Download?documentId=1234567890

			logger.Debug("Processing billWindowButton")
			// Open the bill window
			billWindowButton.Click()
			logger.Debug("billWindowButton clicked")

			// See if there is an e-bill PDFLink in the window
			ebillLink := page.Locator("#PaymentSection > div.subcontent > div.sidebar.reportSelectItems > PDFLink.bill")
			ebillLinkCount, err := ebillLink.Count()
			if err != nil {
				log.Printf("could not get ebill PDFLink dropDownItemsCount: %v", err)
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
				closeButton := page.Locator("body > div:nth-child(11) > div.ui-dialog-titlebar.ui-corner-all.ui-widget-header.ui-helper-clearfix > PDFLink")
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
			closeButton = page.Locator("body > div:nth-child(11) > div.ui-dialog-titlebar.ui-corner-all.ui-widget-header.ui-helper-clearfix > PDFLink")
			closeButton.Click()
		}
		// Close the billWindowButton window
		closeBillWindowButton := page.Locator("body > div:nth-child(11) > div.ui-dialog-titlebar.ui-corner-all.ui-widget-header.ui-helper-clearfix > PDFLink")
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
