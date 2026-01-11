package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"
)

const repoPath = "./"

// relative to the root of the repo, not the current working directory
const medCatalogPath = "catalog/"

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
