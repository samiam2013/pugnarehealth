package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/template"
)

const repoPath = "/Users/sam/git/samiam2013/pugnarehealth/"

// relative to the root of the repo, not the current working directory
const medCatalogPath = "catalog/"

func main() {
	fmt.Println("starting pugnare.health...")
	products, err := getCatalog(medCatalogPath)
	if err != nil {
		fmt.Println("Error getting catalog:", err)
		os.Exit(1)
	}

	// for _, p := range products {
	// 	fmt.Printf("Product: %#v\n", p)
	// }

	if err = renderIndex(products); err != nil {
		fmt.Println("Error rendering index:", err)
		os.Exit(1)
	}
}

type product struct {
	IngredientName string `json:"ingredient_name"`
	BrandName      string `json:"brand_name,omitempty"`
	MedicineType   string `json:"medicine_type,omitempty"`
	AdminRoute     string `json:"administration_route,omitempty"`
	DoseFrequency  string `json:"dose_frequency,omitempty"`
	Savings        string `json:"savings,omitempty"`
	Phone          string `json:"phone,omitempty"`
	Link           string `json:"link,omitempty"`
	ColorClass     string `json:"color_class,omitempty"`
}

func getCatalog(path string) ([]product, error) {
	// get the list of files in that folder, accumulate files that end in .json
	files := []string{}
	entries, err := os.ReadDir(repoPath + medCatalogPath)
	if err != nil {
		return []product{}, errors.Join(err, errors.New("reading catalog directory"))
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
			return []product{}, errors.Join(err, fmt.Errorf("reading file %s", file))
		}

		var p product
		if err = json.Unmarshal(content, &p); err != nil {
			return []product{}, errors.Join(err, fmt.Errorf("parsing JSON in file %s", file))
		}
		products = append(products, p)
	}

	return products, nil
}

func renderIndex(products []product) error {
	// open the index.gohtml file, read its content
	content, err := os.ReadFile(repoPath + "index.gohtml")
	if err != nil {
		return errors.Join(err, errors.New("reading index.gohtml"))
	}
	indexTemplate := string(content)

	funcMap := template.FuncMap{
		"hasPrefix": strings.HasPrefix,
	}

	t, err := template.New("index").Funcs(funcMap).Parse(indexTemplate)
	if err != nil {
		return errors.Join(err, errors.New("parsing index.gohtml template"))
	}

	data := struct {
		Products []product
	}{
		Products: products,
	}

	outputFile, err := os.Create(repoPath + "public/index.html")
	if err != nil {
		return errors.Join(err, errors.New("creating index.html"))
	}
	defer outputFile.Close()

	if err = t.Execute(outputFile, data); err != nil {
		return errors.Join(err, errors.New("executing template for index.html"))
	}

	return nil
}
