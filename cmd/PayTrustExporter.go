package main

import (
	"PayTrust/onepassword"
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/playwright-community/playwright-go"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// lvl is global, so we can set it from the command line
var lvl = new(slog.LevelVar)

// logger is global, so we can use it everywhere in the script
var logger *slog.Logger

// Main entry point for execution
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
	userNameInput := page.Locator("#UserName input[type=text]")

	// Get the count to see if we found it ...
	var UserNameInputCount int

	// See if we found it ...
	UserNameInputCount, err = userNameInput.Count()
	if err != nil {
		log.Fatalf("could not get passwordInput: %v", err)
	}
	// If we didn't find it, we are done
	if UserNameInputCount == 0 {
		log.Fatalf("could not find passwordInput")
	}

	// Fill in the username
	userNameInput.Fill(*username)
	//log.Printf("Element: %#v", passwordInput)
	logger.Debug(fmt.Sprintf("UserName filled in %#v", *username))

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
	err = continueButton.Click()
	if err != nil {
		log.Fatalf("could not click continueButton: %v", err)
	}
	logger.Debug("Continue button clicked")
	err = page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State:   playwright.LoadStateDomcontentloaded,
		Timeout: playwright.Float(3000),
	})
	if err != nil {
		log.Fatalf("could not WaitForLoadState: %v", err)
	}

	authForm := page.Locator("div.page.authentication > div.region.right > form")
	err = authForm.WaitFor(playwright.LocatorWaitForOptions{
		Timeout: playwright.Float(3000),
	})
	if err != nil {
		log.Fatalf("could not WaitFor authForm: %v", err)
	}
	// See if we need to do phone verification
	formCount, err := authForm.Count()
	if err != nil {
		logger.Error("Could not get form: %v", err)
	}
	// On the password page, there are two forms. On the phone verification page, there is one
	if formCount == 1 {
		formHTML, err := authForm.Evaluate("el => el.outerHTML", nil)
		if err != nil {
			log.Fatalf("could not get html: %v", err)
		}
		// This is where the phone call could happen ...
		// https://login.billscenter.paytrust.com/3004/OOBA/Preview
		if strings.Contains(fmt.Sprintf("%v", formHTML), "OOBA/Preview") {
			logger.Info("Handling the logic to for phone verification")
			// We need to wait for the phone call to finish
			ManualStepCompletion("Phone call received")
		}
	}

	// Enter password
	passwordInput := page.Locator("#Password")
	passwordInput.WaitFor()
	//log.Printf("Element: %#v", passwordInput)
	var passwordInputCount int
	passwordInputCount, err = passwordInput.Count()
	if err != nil {
		log.Fatalf("could not get passwordInput: %v", err)
	}
	if passwordInputCount == 0 {
		log.Fatalf("could not find passwordInput")
	}
	passwordInput.Fill(*password)
	logger.Debug("Password filled in")

	// Click continue
	// Selector from browser: body > div > div.region.right > form:nth-child(3) > div.buttons > button.button.primary
	// TODO: Find a better way to do this (I feel like this is brittle)
	signonButton := page.Locator("body > div > div.region.right > form:nth-child(3) > div.buttons > button.button.primary")
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
	err = page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateDomcontentloaded,
	})
	if err != nil {
		log.Fatalf("could not WaitForLoadState from signon button: %v", err)
	}

	// Should be logged in now
	// 	// body > div.ui-dialog.ui-corner-all.ui-widget.ui-widget-content.ui-front.EPP > div.ui-dialog-titlebar.ui-corner-all.ui-widget-header.ui-helper-clearfix > button
	noticeClear := page.Locator(`button[title="Close"]`)
	err = noticeClear.WaitFor()
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
	reportOptions.WaitFor()
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
	logger.Debug(fmt.Sprintf("dropDownItems: %#v", dropDownItemsCount))

	// Get the drop down items
	reportSelectItems, err := dropDownItems.All()
	if err != nil {
		log.Fatalf("could not get reportSelectItems: %v", err)
	}
	logger.Debug(fmt.Sprintf("reportSelectItems: %#v", reportSelectItems))

	var foundReport bool
	// All dates value
	allDates := "Include All Dates"
	var text string
	for _, item := range reportSelectItems {
		text, err = item.TextContent()
		if err != nil {
			log.Fatalf("could not get text: %v", err)
		}
		if strings.Contains(text, allDates) {
			logger.Debug("Found All Dates")
			foundReport = true
			reportTitleSelector := "#Reports_ViewReportViewDiv > div.container.sectionsContainer > div.section.content.contentSection.clear > div.report-title > h1"
			reportTitle := page.Locator(reportTitleSelector)
			err = reportTitle.WaitFor(playwright.LocatorWaitForOptions{
				State:   playwright.WaitForSelectorStateAttached,
				Timeout: playwright.Float(10000),
			})
			_, err = reportTitle.Evaluate("el => el.remove()", nil)
			if err != nil {
				log.Fatalf("could not remove reportTitle: %v", err)
			}

			err = item.Click()
			if err != nil {
				log.Fatalf("could not click All Dates: %v", err)
			}

			reportTitleSelector = fmt.Sprintf(reportTitleSelector)
			err = page.Locator(reportTitleSelector).WaitFor(playwright.LocatorWaitForOptions{
				State:   playwright.WaitForSelectorStateAttached,
				Timeout: playwright.Float(10000),
			})
			if err != nil {
				log.Fatalf("Report took too long to load after clicking All Dates: %v", err)
			}
			logger.Debug("All Dates clicked and title ready")
			break
		}
	}
	// If we didn't find the report, we are done
	if !foundReport {
		log.Fatalf("could not find report for %#v", allDates)
	}

	// Wait for the report to load
	err = page.Locator("tr.total > td.totalLabel > div.totalDiv > span.totalLabel").WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateAttached,
		Timeout: playwright.Float(10000),
	})
	if err != nil {
		log.Fatalf("could not WaitFor totalLabel: %v", err)
	}

	// So now we have the report up, get all the PDF links
	// selector #subcategory\\ Fis\\.Epp\\.DomainModel\\.BillPay\\.ReportGroup > tbody > tr > td.column.bill > billButton
	PDFLinks := page.Locator("tr > td.column.bill > button.billIcon")
	err = PDFLinks.Last().WaitFor(
		playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateAttached,
			Timeout: playwright.Float(10000),
		})
	if err != nil {
		log.Fatalf("could not WaitFor PDFLinks: %v", err)
	}
	var PDFLinksCount int
	PDFLinksCount, err = PDFLinks.Count()
	if err != nil {
		log.Fatalf("could not get PDFLinksCount of links: %v", err)
	}
	log.Printf("PDFLinksCount: %#v", PDFLinksCount)
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
	logger.Debug(fmt.Sprintf("PDFLinksAll: %#v", len(PDFLinksAll)))

	var NewPages []playwright.Page
	// Loop through all the links ....
	for _, PDFLink := range PDFLinksAll {
		var html interface{}
		html, err = PDFLink.Evaluate("el => el.outerHTML", nil)
		if err != nil {
			log.Fatalf("could not get html: %v", err)
		}
		logger.Debug(fmt.Sprintf("PDFLink html: %#v", html))

		// Open the window ...
		err = PDFLink.Click()
		if err != nil {
			log.Fatalf("could not click PDFLink: %v", err)
		}
		err = page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
			State:   playwright.LoadStateDomcontentloaded,
			Timeout: playwright.Float(30000),
		})
		logger.Debug("PDFLink clicked")

		// Get the close button for the PDF window ...
		// #ViewBills > div.view.extraLarge > div > div.section.buttons.buttonsSection > button
		closeButton := page.Locator("#ViewBills > div.view.extraLarge > div > div.section.buttons.buttonsSection > button")
		err = closeButton.WaitFor()
		if err != nil {
			log.Fatalf("could not WaitFor closeButton: %v", err)
		}

		// Get the selector for the iframe
		// #ViewBills > div.view.extraLarge > div > div.container.sectionsContainer > div.section.content.contentSection.clear > div.area.billselection.clear > div > div.field.billSelection.clear > select
		billSelector := page.Locator("#ViewBills > div.view.extraLarge > div > div.container.sectionsContainer > div.section.content.contentSection.clear > div.area.billselection.clear > div > div.field.billSelection.clear > select")
		var billSelectorReady bool
		billSelectorReady, err = billSelector.IsEnabled(playwright.LocatorIsEnabledOptions{
			Timeout: playwright.Float(10000),
		})
		if err != nil {
			log.Fatalf("could not get billSelectorReady: %v", err)
		}
		if !billSelectorReady {

			var options interface{}

			options, err = billSelector.Evaluate(`element => Array.from(element.options).map(option => option.value)`, nil)
			if err != nil {
				log.Fatalf("could not get select options: %v", err)
			}
			tmpArray := options.([]interface{})
			optionsArray := make([]string, len(tmpArray))
			for i, v := range tmpArray {
				optionsArray[i] = fmt.Sprint(v)
			}
			logger.Debug(fmt.Sprintf("options: %#v", optionsArray))

			for optno, option := range optionsArray {
				var selectedOption []string
				selectedOption, err = billSelector.SelectOption(playwright.SelectOptionValues{Values: &[]string{option}})
				if err != nil {
					log.Fatalf("%d could not select option: %v", optno, err)
				}
				logger.Debug(fmt.Sprintf("Processing selectedOption: %#v", selectedOption))
				frameMe := page.Locator("iframe")
				var frameMeCount int
				frameMeCount, err = frameMe.Count()
				if err != nil {
					log.Fatalf("could not get frameMeCount: %v", err)
				}
				if frameMeCount == 0 {
					err = CloseBillWindow(closeButton)
					if err != nil {
						log.Fatalf("could not click closeButton: %v", err)
					}
					continue
				}
				logger.Debug(fmt.Sprintf("frameMeCount: %#v", frameMeCount))
				html, err = frameMe.GetAttribute("src")
				if err != nil {
					log.Fatalf("could not get html: %v", err)
				}
				logger.Debug(fmt.Sprintf("frameMe html: %v", html))

				var newPage playwright.Page
				newPage, err = browser.NewPage()
				if err != nil {
					log.Fatalf("could not create newPage: %v", err)
				}

				if response, err = newPage.Goto(fmt.Sprintf("%v", html)); err != nil {
					log.Fatalf("could not go to %v: %v", *url, err)
				}
				if response.Status() != 200 {
					log.Fatalf("could not goto: %v", response.Status())
				}

				NewPages = append(NewPages, newPage)
				// for (const node of $('#viewer').shadowRoot.childNodes) { if (node.nodeName === "VIEWER-TOOLBAR") {console.log(node.shadowRoot)} }
				//viewerToolbar := newPage.Locator("#viewer")
				//err = viewerToolbar.WaitFor()
				//if err != nil {
				//	log.Fatalf("could not WaitFor viewerToolbar: %v", err)
				//}
				//html, err = viewerToolbar.Evaluate("el => el.shadowRoot.childNodes", nil)
				//if err != nil {
				//	log.Fatalf("could not get html: %v", err)
				//}
				//
				//logger.Debug(fmt.Sprintf("newPage html: %#v", html))
				// #ViewBills > div.view.extraLarge > div > div.container.sectionsContainer > div.section.content.contentSection.clear > div.area.billimage.clear > div.areaContent
				// Get the displayArea item
				//displayArea := page.FrameLocator("iframe")
				//embed := displayArea.Locator("body")
				//html, err = embed.InnerHTML()
				//if err != nil {
				//	log.Fatalf("could not get html: %v", err)
				//}

				// Get image billButton
				billNewWindowLink := page.Locator("#ViewBills > div.view.extraLarge > div > div.container.sectionsContainer > div.section.content.contentSection.clear > div.area.billimage.clear > div.areaHeader > span.newWindow > a")
				err = billNewWindowLink.WaitFor(playwright.LocatorWaitForOptions{
					Timeout: playwright.Float(5000),
				})
				if err != nil {
					log.Printf("could not WaitFor billNewWindowLink: %v", err)
					err = CloseBillWindow(closeButton)
					if err != nil {
						logger.Error(fmt.Sprintf("could not click closeButton: %v", err))
					}
					continue
				}
				var innerHTML string
				innerHTML, err = billNewWindowLink.InnerHTML()
				if err != nil {
					log.Printf("could not get innerHTML: %v", err)
					err = CloseBillWindow(closeButton)
					continue
				}
				logger.Debug(fmt.Sprintf("innerHTML: %#v", innerHTML))
				var billNewWindowLinkCount int
				billNewWindowLinkCount, err = billNewWindowLink.Count()
				if err != nil {
					logger.Error(fmt.Sprintf("could not get image billNewWindowLinkCount: %v", err))
					err = CloseBillWindow(closeButton)
					if err != nil {
						log.Fatalf("could not click closeButton: %v", err)
					}
					continue
				}
				// Has an image PDFLink in the window, so click i
				if billNewWindowLinkCount == 1 {
					var outerHtml interface{}
					outerHtml, err = billNewWindowLink.Evaluate("el => el.outerHTML", nil, playwright.LocatorEvaluateOptions{Timeout: playwright.Float(30000)})
					if err != nil {
						logger.Error(fmt.Sprintf("could not get image billNewWindowLink outerHTML: %v", err))
						err = CloseBillWindow(closeButton)
						if err != nil {
							logger.Error(fmt.Sprintf("could not click closeButton: %v", err))
						}
						continue
					}
					linkText := fmt.Sprintf("%v", outerHtml)

					// <a href="([^"])+
					re := regexp.MustCompile(`<a href="([^"]*?)"`)
					res := re.FindAllStringSubmatch(linkText, 1)
					log.Printf("linkText href: %#v", res[0][1])

					page.OnDialog(func(dialog playwright.Dialog) {
						// Get dialog path
						content, err := dialog.Page().Content()
						if err != nil {
							log.Fatalf("could not get dialog path: %v", err)
						}
						logger.Debug("dialog content: %#v", content)
					})
					// Click the open in new window link ...
					err = billNewWindowLink.Click()
					if err != nil {
						log.Fatalf("could not click billNewWindowLink: %v", err)
					}
					var popup playwright.Page
					popup, err = page.ExpectPopup(func() error {
						logger.Debug("ExpectPopup")
						return nil
					},
					)
					if err != nil {
						log.Fatalf("could not get popup: %v", err)
					}

					html, err = popup.Content()
					if err != nil {
						log.Fatalf("could not get content: %v", err)
					}
					logger.Debug(fmt.Sprintf("popup html: %#v", html))

					err = CloseBillWindow(closeButton)
					if err != nil {
						logger.Error(fmt.Sprintf("could not click closeButton: %v", err))
					}
					err = popup.Close()
					if err != nil {
						log.Fatalf("could not close popup: %v", err)
					}
				}
			}
			log.Printf("%d NewPages", len(NewPages))
			// Close the billNewWindowLink window
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
}

func CloseBillWindow(closeButton playwright.Locator) (err error) {
	// Close the dialog for the bill ...
	err = closeButton.WaitFor()
	if err != nil {
		logger.Error(fmt.Sprintf("could not WaitFor closeButton: %v", err))
		return
	}
	err = closeButton.Click()
	return
}

// GetPDF - get the PDF from the URL
func GetPDF(html string) (PDFbody []byte, err error) {
	var resp *http.Response

	// Set up HTTP client
	resp, err = http.Get(fmt.Sprintf("%v", html))
	if err != nil {
		logger.Error(fmt.Sprintf("could not get PDF: %v", err))
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Fatalf("could not get PDF: %v", resp.StatusCode)
	}
	PDFbody, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("could not read PDF body: %v", err)
	}

	return

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
	logger.Debug(fmt.Sprintf("vaultUserName: %v", vaultUserName))
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
