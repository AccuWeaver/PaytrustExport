# Paytrust Export

This is a script to scrape all the PDFs from Paytrust and save them to a folder.

Running the script with no arguments, will use the defaults and look up your Paytrust username and password in 1Password.
It assumes that the entry is in you "Personal" vault and has the tag "Paytrust" (and of course is the only entry with
that tag in your personal vault).

You can run without 1Password by specifying the username and password on the command line with the `-username` and
`-password` flags.

It also assumes that you have a report named "Include All Dates" that will return all the bills.  You can specify a
different report name with the `-reportName` flag.

It brute forces a few things, and is a bit fragile, but it worked for me to pull a few years of bills in PDF format
from Paytrust.

## Command Line Usage

```bash
/opt/homebrew/opt/go/libexec/bin/go build -o /Users/robweaver/Library/Caches/JetBrains/IntelliJIdea2023.3/tmp/GoLand/___go_build_PayTrustExporter_go /Users/Shared/Projects/PaytrustExport/cmd/PayTrustExporter.go #gosetup
/Users/robweaver/Library/Caches/JetBrains/IntelliJIdea2023.3/tmp/GoLand/___go_build_PayTrustExporter_go --help
Usage of /Users/robweaver/Library/Caches/JetBrains/IntelliJIdea2023.3/tmp/GoLand/___go_build_PayTrustExporter_go:
  -debug
        Debug flag
  -password string
        Password to use if not using 1Password
  -reportName string
        Report name for Paytrust to get all the bills (default "Include All Dates")
  -tags string
        1Password tags (default "Paytrust")
  -url string
        Paytrust login URL (default "https://login.billscenter.paytrust.com/3004/")
  -username string
        Paytrust login username
  -vault string
        1Password vault (default "Personal")
```


