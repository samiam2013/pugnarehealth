package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

const repoPath = "./"

// relative to the root of the repo, not the current working directory
const medCatalogPath = "catalog/"

var medTypes = map[string]struct{}{
	"Continuous Glucose Monitor": {},
	"Manual Insulin Pump":        {},
	"SGLT-2 Inhibitor":           {},
	"GLP-1 Agonist":              {},
	"DPP-4 Inhibitor":            {},
	"GLP-1/GIP Dual Agonist":     {},
}

var adminRoutes = map[string]struct{}{
	"Oral Tablet":            {},
	"Subcutaneous Injection": {},
	"Automatic Applicator":   {},
}

var savingsTypes = map[string]struct{}{
	"Copay Discount Card":                {},
	"Patient Assistance Program":         {},
	"Medicare Prescription Payment Plan": {},
}

func main() {
	var skipUpdateCheck bool
	flag.BoolVar(&skipUpdateCheck, "skip-update-check", false, "Skip checking for FDA label updates using the OpenFDA API")
	flag.Parse()

	fmt.Println("starting webserver for pugnare.health")
	products, err := getCatalog(medCatalogPath)
	if err != nil {
		fmt.Println("Error getting catalog:", err)
		os.Exit(1)
	}

	if !skipUpdateCheck {
		brandNames := []string{}
		for _, p := range products {
			if p.AdminRoute == "Automatic Applicator" {
				continue // skip CGMs, pumps, etc without FDA labels
			}
			brandNames = append(brandNames, p.BrandName)
		}

		recencyResults, err := fdaLabelRecencyLookup(brandNames)
		if err != nil {
			fmt.Println("Error looking up FDA label recency:", err)
			os.Exit(1)
		}

		// print out the results
		for i, p := range products {
			recency, ok := recencyResults[p.BrandName]
			if !ok {
				fmt.Printf("No FDA label recency found for %s\n", p.BrandName)
				continue
			}
			lastUpdated, err := time.Parse("2006-01-02", p.FDALabelUpdated)
			if err != nil {
				fmt.Printf("Error parsing existing FDA label updated date for %s: %v\n", p.BrandName, err)
				continue
			}
			if recency.After(lastUpdated) {
				fmt.Printf("FDA label for %s has been updated since last recorded date. New effective date: %s (was %s)\n",
					p.BrandName, recency.Format("2006-01-02"), lastUpdated.Format("2006-01-02"))
				products[i].FDALabelNeedsUpdate = true
			}
		}
	}

	phoneRe := regexp.MustCompile(`^1-\d{3}-\d{3}-\d{4}$`)
	// validate the products
	for _, p := range products {
		// Check that the unconstrained fields are not empty
		if strings.TrimSpace(p.BrandName) == "" {
			fmt.Printf("Failed: Brand name is empty for ingredient '%s'\n", p.IngredientName)
			os.Exit(1)
		}
		if strings.TrimSpace(p.IngredientName) == "" {
			fmt.Printf("Failed: Ingredient name is empty for brand '%s'\n", p.BrandName)
			os.Exit(1)
		}
		if strings.TrimSpace(p.DoseFrequency) == "" {
			fmt.Printf("Failed: Dose frequency is empty for product '%s'\n", p.BrandName)
			os.Exit(1)
		}
		if len(p.Savings) == 0 {
			fmt.Printf("Failed: Savings information is empty for product '%s'\n", p.BrandName)
			os.Exit(1)
		}

		// check the medicine type is one in the list
		if _, ok := medTypes[p.MedicineType]; !ok {
			fmt.Printf("Failed: Medicine type '%s' for product '%s' is not in the recognized list\n", p.MedicineType, p.BrandName)
			fmt.Printf("Recognized medicine types are:\n")
			for k := range medTypes {
				fmt.Printf(" - %s\n", k)
			}
			os.Exit(1)
		}

		// check the administration route is one in the list
		if _, ok := adminRoutes[p.AdminRoute]; !ok {
			fmt.Printf("Failed: Administration route '%s' for product '%s' is not in the recognized list\n", p.AdminRoute, p.BrandName)
			fmt.Printf("Recognized administration routes are:\n")
			for k := range adminRoutes {
				fmt.Printf(" - %s\n", k)
			}
			os.Exit(1)
		}

		// validate each savings program's phone and link
		for _, sp := range p.Savings {
			if strings.TrimSpace(sp.Description) == "" {
				fmt.Printf("Failed: Savings program description is empty for product '%s'\n", p.BrandName)
				os.Exit(1)
			}
			if strings.TrimSpace(sp.Phone) != "" {
				if !phoneRe.MatchString(sp.Phone) {
					fmt.Printf("Failed: Phone number '%s' for product '%s' is not in the format 1-800-555-5555\n", sp.Phone, p.BrandName)
					os.Exit(1)
				}
			}
			if strings.TrimSpace(sp.Link) != "" {
				if !strings.HasPrefix(sp.Link, "http://") && !strings.HasPrefix(sp.Link, "https://") {
					fmt.Printf("Failed: Link '%s' for product '%s' is not a valid URL (must start with http:// or https://)\n", sp.Link, p.BrandName)
					os.Exit(1)
				}
			}
			if _, ok := savingsTypes[sp.Type]; !ok {
				fmt.Printf("Failed: Savings program type '%s' for product '%s' is not in the recognized list\n", sp.Type, p.BrandName)
				fmt.Printf("Recognized savings program types are:\n")
				for k := range savingsTypes {
					fmt.Printf(" - %s\n", k)
				}
				os.Exit(1)
			}
		}

		// if there is an fda label link
		if strings.TrimSpace(p.FDALabelFile) != "" {
			// make sure the updated date is in YYYY-MM-DD format
			updateTime, err := time.Parse("2006-01-02", p.FDALabelUpdated)
			if err != nil {
				fmt.Printf("Failed: FDA label updated date '%s' for product '%s' is not in YYYY-MM-DD format\n", p.FDALabelUpdated, p.BrandName)
				os.Exit(1)
			}
			// it's impossible to have updated the label in the future
			if updateTime.After(time.Now()) {
				fmt.Printf("Failed: FDA label updated date '%s' for product '%s' is in the future\n", p.FDALabelUpdated, p.BrandName)
				os.Exit(1)
			}
			// make sure the link is to FDA's label repository
			if !strings.HasPrefix(p.FDALabelFile, "https://www.accessdata.fda.gov/drugsatfda_docs/label/") {
				fmt.Printf("Failed: FDA label file link '%s' for product '%s' is not a valid FDA label repository URL\n", p.FDALabelFile, p.BrandName)
				os.Exit(1)
			}
			// make sure the link is valid (parses as a URL)
			u, err := url.ParseRequestURI(p.FDALabelFile)
			if err != nil {
				fmt.Printf("Failed: FDA label file link '%s' for product '%s' is not a valid URL\n", p.FDALabelFile, p.BrandName)
				os.Exit(1)
			}
			// it has to be a link to a PDF
			if !strings.HasSuffix(strings.ToLower(u.Path), ".pdf") {
				fmt.Printf("Failed: FDA label file link '%s' for product '%s' is not a link to a PDF file\n", p.FDALabelFile, p.BrandName)
				os.Exit(1)
			}
			// TODO send a HEAD request to make sure the link is reachable?
		}

		// TODO: check/generate css colors/classes from one source?
	}

	if err = renderIndex(products); err != nil {
		fmt.Println("Error rendering index:", err)
		os.Exit(1)
	}
}

type product struct {
	IngredientName      string        `json:"ingredient_name"`
	BrandName           string        `json:"brand_name"`
	MedicineType        string        `json:"medicine_type"`
	AdminRoute          string        `json:"administration_route"`
	DoseFrequency       string        `json:"dose_frequency,omitempty"`
	Savings             []savingsInfo `json:"savings"`
	FDALabelFile        string        `json:"fda_label_file,omitempty"`
	FDALabelUpdated     string        `json:"fda_label_file_updated,omitempty"` // YYYY-MM-DD
	FDALabelNeedsUpdate bool          `json:"fda_label_needs_update,omitempty"`
	ColorClass          string        `json:"color_class,omitempty"`
}

type savingsInfo struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Phone       string `json:"phone,omitempty"`
	Link        string `json:"link,omitempty"`
	Eligibility struct {
		PrivateInsurance    bool     `json:"private_insurance,omitempty"`
		GovernmentInsurance bool     `json:"government_insurance,omitempty"`
		CashPay             bool     `json:"cash_pay,omitempty"`
		OtherCriteria       []string `json:"other_criteria,omitempty"`
	} `json:"eligibility,omitempty"`
}

func getCatalog(path string) ([]product, error) {
	// get the list of files in that folder, accumulate files that end in .json
	files := []string{}
	entries, err := os.ReadDir(repoPath + medCatalogPath)
	if err != nil {
		return []product{}, errors.Join(errors.New("failed reading catalog directory"), err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			files = append(files, entry.Name())
		}
	}
	fmt.Printf("Found %d JSON files in %s\n", len(files), path)
	// for each file, read and parse the JSON into a product struct, accumulate into a slice
	products := []product{}
	for _, file := range files {
		// fmt.Printf("Processing file: %s\n", file)
		content, err := os.ReadFile(repoPath + medCatalogPath + file)
		if err != nil {
			return []product{}, errors.Join(errors.New("failed reading file "+file), err)
		}

		var p product
		if err = json.Unmarshal(content, &p); err != nil {
			return []product{}, errors.Join(errors.New("failed parsing JSON in file "+file), err)
		}
		products = append(products, p)
	}

	return products, nil
}

func renderIndex(products []product) error {
	// open the index.gohtml file, read its content
	content, err := os.ReadFile(repoPath + "index.gohtml")
	if err != nil {
		return errors.Join(errors.New("failed reading index.gohtml"), err)
	}
	indexTemplate := string(content)

	funcMap := template.FuncMap{
		"hasPrefix": strings.HasPrefix,
		"truncate": func(s string, n int) string {
			if len(s) <= n {
				return s
			}
			return s[:n] + "..."
		},
		"subtract": func(a, b int) int {
			return a - b
		},
	}

	t, err := template.New("index").Funcs(funcMap).Parse(indexTemplate)
	if err != nil {
		return errors.Join(errors.New("failed parsing index.gohtml template"), err)
	}

	data := struct {
		Products []product
	}{
		Products: products,
	}

	outputFile, err := os.Create(repoPath + "public/index.html")
	if err != nil {
		return errors.Join(errors.New("failed creating index.html"), err)
	}
	defer outputFile.Close()

	if err = t.Execute(outputFile, data); err != nil {
		return errors.Join(errors.New("failed executing template for index.html"), err)
	}

	return nil
}
