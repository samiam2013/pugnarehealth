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
	"slices"
	"sort"
	"strings"
	"time"
)

const repoPath = "./"

// relative to the root of the repo, not the current working directory
const medCatalogPath = "catalog/"

var medTypeEnum = NewEnum([]string{
	"CGM",
	"SGLT-2",
	"GLP-1",
	"DPP-4",
	"Insulin Delivery System",
	"Insulin",
})

var adminRouteEnum = NewEnum([]string{
	"Oral Tablet",
	"Subcutaneous Injection",
	"Automatic Applicator",
	"Tubeless Insulin Pump",
})

var savingsTypeEnum = NewEnum([]string{
	"Copay Discount Card",
	"Patient Assistance Program",
	"Medicare Prescription Payment Plan",
	"Free Trial Offer",
})

func main() {
	var skipUpdateCheck bool
	flag.BoolVar(&skipUpdateCheck, "skip-update-check", false, "Skip checking for FDA label updates using the OpenFDA API")
	flag.Parse()

	fmt.Println("starting render...")
	products, err := getCatalog(medCatalogPath)
	if err != nil {
		fmt.Println("Error getting catalog:", err)
		os.Exit(1)
	}

	if !skipUpdateCheck {
		if err = checkForLabelUpdates(products); err != nil {
			fmt.Println("Error checking for FDA label updates:", err)
			os.Exit(1)
		}
	}

	// validate the products
	for _, p := range products {
		if err = p.Validate(); err != nil {
			fmt.Printf("Validation error for product %s: %v\n", p.BrandName, err)
			os.Exit(1)
		}

		// TODO: check/generate css colors/classes from one source?
	}

	// sort the products by ListPosition
	// products with ListPosition 0 (not set) go to the end
	sortedProducts := []product{}
	unsortedProducts := []product{}
	for _, p := range products {
		if p.ListPosition > 0 {
			sortedProducts = append(sortedProducts, p)
		} else {
			unsortedProducts = append(unsortedProducts, p)
		}
	}
	// sort the sortedProducts slice
	sp := byListPosition(sortedProducts)
	sort.Sort(sp)
	// append the unsorted products to the end
	sortedProducts = append(sp, unsortedProducts...)
	products = sortedProducts

	if err = renderIndex(products); err != nil {
		fmt.Println("Error rendering index:", err)
		os.Exit(1)
	}
}

func validateFDALabelLink(p product) error {
	// make sure the updated date is in YYYY-MM-DD format
	updateTime, err := time.Parse("2006-01-02", p.FDALabelUpdated)
	if err != nil {
		return errors.Join(fmt.Errorf("Failed: FDA label updated date '%s' for product '%s' is not in YYYY-MM-DD format\n", p.FDALabelUpdated, p.BrandName, err))
	}
	// it's impossible to have updated the label in the future
	if updateTime.After(time.Now()) {
		return fmt.Errorf("Failed: FDA label updated date '%s' for product '%s' is in the future", p.FDALabelUpdated, p.BrandName)
	}
	// make sure the link is to FDA's label repository
	if !strings.HasPrefix(p.FDALabelFile, "https://www.accessdata.fda.gov/drugsatfda_docs/label/") {
		return fmt.Errorf("Failed: FDA label file link '%s' for product '%s' is not a valid FDA label repository URL", p.FDALabelFile, p.BrandName)
	}
	// make sure the link is valid (parses as a URL)
	u, err := url.ParseRequestURI(p.FDALabelFile)
	if err != nil {
		return fmt.Errorf("Failed: FDA label file link '%s' for product '%s' is not a valid URL", p.FDALabelFile, p.BrandName)
	}
	// it has to be a link to a PDF
	if !strings.HasSuffix(strings.ToLower(u.Path), ".pdf") {
		return fmt.Errorf("Failed: FDA label file link '%s' for product '%s' is not a link to a PDF file", p.FDALabelFile, p.BrandName)
	}
	// TODO send a HEAD request to make sure the link is reachable?
	return nil
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
	ListPosition        int           `json:"list_position,omitempty"`
}

func (p product) Validate() error {
	// Check that the unconstrained fields are not empty
	if slices.Contains([]string{p.BrandName, p.IngredientName, p.DoseFrequency}, "") {
		return fmt.Errorf("Failed: Brand name, ingredient name, and dose frequency cannot be empty for product '%s'", p.BrandName)
	}
	if len(p.Savings) == 0 {
		return fmt.Errorf("Failed: Savings information is empty for product '%s'", p.BrandName)
	}

	// check the medicine type is one in the list
	if err := medTypeEnum.CheckError(p.MedicineType); err != nil {
		return errors.Join(fmt.Errorf("Failed: Medicine type '%s' for product '%s' is invalid. ", p.MedicineType, p.BrandName), err)
	}

	// check the administration route is one in the list
	if err := adminRouteEnum.CheckError(p.AdminRoute); err != nil {
		return errors.Join(fmt.Errorf("Failed: Administration route '%s' for product '%s' is invalid. ", p.AdminRoute, p.BrandName), err)
	}

	// validate each savings program's phone and link
	for _, s := range p.Savings {
		if err := s.Validate(); err != nil {
			return fmt.Errorf("Failed: Savings program '%s' for product '%s' is invalid: %v", s.Description, p.BrandName, err)
		}
	}

	// if there is an fda label link, validate it
	if strings.TrimSpace(p.FDALabelFile) != "" {
		if err := validateFDALabelLink(p); err != nil {
			return fmt.Errorf("Failed: FDA label validation for product '%s': %v", p.BrandName, err)
		}
	}

	return nil
}

// make the sort functions for products based on ListPosition
type byListPosition []product

func (a byListPosition) Len() int           { return len(a) }
func (a byListPosition) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byListPosition) Less(i, j int) bool { return a[i].ListPosition < a[j].ListPosition }

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

func (s savingsInfo) Validate() error {
	phoneRe := regexp.MustCompile(`^1-\d{3}-\d{3}-\d{4}$`)
	if strings.TrimSpace(s.Description) == "" {
		panic("Savings description cannot be empty for product " + s.Description)
	}
	if strings.TrimSpace(s.Phone) != "" {
		if !phoneRe.MatchString(s.Phone) {
			panic(fmt.Sprintf("Phone number '%s' is not in the format 1-800-555-5555", s.Phone))
		}
	}
	if strings.TrimSpace(s.Link) != "" {
		if !strings.HasPrefix(s.Link, "http://") && !strings.HasPrefix(s.Link, "https://") {
			fmt.Printf("Failed: Link '%s' for product '%s' is not a valid URL (must start with http:// or https://)\n", s.Link, s.Description)
			os.Exit(1)
		}
	}
	if err := savingsTypeEnum.CheckError(s.Type); err != nil {
		panic(err)
	}

	return nil
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

	fmt.Println(outputFile.Name() + " rendered successfully.")

	return nil
}
