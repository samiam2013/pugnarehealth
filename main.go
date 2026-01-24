package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"
)

const repoPath = "./"

// relative to the root of the repo, not the current working directory
const medCatalogPath = "catalog/"

var medTypes = map[string]struct{}{
	"Continuous Glucose Monitor": {},
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

func main() {
	var skipUpdateCheck bool
	flag.BoolVar(&skipUpdateCheck, "skip-update-check", false, "Skip checking for FDA label updates using the OpenFDA API")
	flag.Parse()

	fmt.Println("starting pugnare.health...")
	products, err := getCatalog(medCatalogPath)
	if err != nil {
		fmt.Println("Error getting catalog:", err)
		os.Exit(1)
	}

	if !skipUpdateCheck {
		brandNames := []string{}
		for _, p := range products {
			if p.MedicineType == "Continuous Glucose Monitor" {
				continue // skip CGMs for now
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
		if strings.TrimSpace(p.Savings) == "" {
			fmt.Printf("Failed: Savings info is empty for product '%s'\n", p.BrandName)
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

		// make sure the phone number is matches 1-800-555-5555 format if not empty
		if strings.TrimSpace(p.Phone) != "" {
			if !phoneRe.MatchString(p.Phone) {
				fmt.Printf("Failed: Phone number '%s' for product '%s' is not in the format 1-800-555-5555\n", p.Phone, p.BrandName)
				os.Exit(1)
			}
		}

		// make sure the link is a valid URL if not empty
		if strings.TrimSpace(p.Link) != "" {
			if !strings.HasPrefix(p.Link, "http://") && !strings.HasPrefix(p.Link, "https://") {
				fmt.Printf("Failed: Link '%s' for product '%s' is not a valid URL (must start with http:// or https://)\n", p.Link, p.BrandName)
				os.Exit(1)
			}
		}

		// TODO: check/generate css colors/classes from one source?
	}

	if err = renderIndex(products); err != nil {
		fmt.Println("Error rendering index:", err)
		os.Exit(1)
	}
}

type product struct {
	IngredientName      string `json:"ingredient_name"`
	BrandName           string `json:"brand_name,omitempty"`
	MedicineType        string `json:"medicine_type,omitempty"`
	AdminRoute          string `json:"administration_route,omitempty"`
	DoseFrequency       string `json:"dose_frequency,omitempty"`
	Savings             string `json:"savings,omitempty"`
	Phone               string `json:"phone,omitempty"`
	Link                string `json:"link,omitempty"`
	FDALabelFile        string `json:"fda_label_file,omitempty"`
	FDALabelUpdated     string `json:"fda_label_file_updated,omitempty"` // YYYY-MM-DD
	FDALabelNeedsUpdate bool   `json:"fda_label_needs_update,omitempty"`
	ColorClass          string `json:"color_class,omitempty"`
}

func getCatalog(path string) ([]product, error) {
	// get the list of files in that folder, accumulate files that end in .json
	files := []string{}
	entries, err := os.ReadDir(repoPath + medCatalogPath)
	if err != nil {
		return []product{}, errors.Join(errors.New("failed reading catalog directory"), err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && len(entry.Name()) > 5 && strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
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
