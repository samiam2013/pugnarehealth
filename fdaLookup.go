package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

type fdaLabelResult struct {
	SplProductDataElements                               []string `json:"spl_product_data_elements"`
	RecentMajorChanges                                   []string `json:"recent_major_changes"`
	RecentMajorChangesTable                              []string `json:"recent_major_changes_table"`
	BoxedWarning                                         []string `json:"boxed_warning"`
	IndicationsAndUsage                                  []string `json:"indications_and_usage"`
	DosageAndAdministration                              []string `json:"dosage_and_administration"`
	DosageFormsAndStrengths                              []string `json:"dosage_forms_and_strengths"`
	Contraindications                                    []string `json:"contraindications"`
	WarningsAndCautions                                  []string `json:"warnings_and_cautions"`
	AdverseReactions                                     []string `json:"adverse_reactions"`
	AdverseReactionsTable                                []string `json:"adverse_reactions_table"`
	DrugInteractions                                     []string `json:"drug_interactions"`
	UseInSpecificPopulations                             []string `json:"use_in_specific_populations"`
	Pregnancy                                            []string `json:"pregnancy"`
	PediatricUse                                         []string `json:"pediatric_use"`
	GeriatricUse                                         []string `json:"geriatric_use"`
	Overdosage                                           []string `json:"overdosage"`
	Description                                          []string `json:"description"`
	ClinicalPharmacology                                 []string `json:"clinical_pharmacology"`
	MechanismOfAction                                    []string `json:"mechanism_of_action"`
	Pharmacodynamics                                     []string `json:"pharmacodynamics"`
	Pharmacokinetics                                     []string `json:"pharmacokinetics"`
	NonclinicalToxicology                                []string `json:"nonclinical_toxicology"`
	CarcinogenesisAndMutagenesisAndImpairmentOfFertility []string `json:"carcinogenesis_and_mutagenesis_and_impairment_of_fertility"`
	ClinicalStudies                                      []string `json:"clinical_studies"`
	ClinicalStudiesTable                                 []string `json:"clinical_studies_table"`
	HowSupplied                                          []string `json:"how_supplied"`
	HowSuppliedTable                                     []string `json:"how_supplied_table"`
	StorageAndHandling                                   []string `json:"storage_and_handling"`
	InformationForPatients                               []string `json:"information_for_patients"`
	SplMedguide                                          []string `json:"spl_medguide"`
	SplMedguideTable                                     []string `json:"spl_medguide_table"`
	InstructionsForUse                                   []string `json:"instructions_for_use"`
	InstructionsForUseTable                              []string `json:"instructions_for_use_table"`
	PackageLabelPrincipalDisplayPanel                    []string `json:"package_label_principal_display_panel"`
	SetID                                                string   `json:"set_id"`
	ID                                                   string   `json:"id"`
	EffectiveTime                                        string   `json:"effective_time"`
	Version                                              string   `json:"version"`
	Openfda                                              struct {
		ApplicationNumber  []string `json:"application_number"`
		BrandName          []string `json:"brand_name"`
		GenericName        []string `json:"generic_name"`
		ManufacturerName   []string `json:"manufacturer_name"`
		ProductNdc         []string `json:"product_ndc"`
		ProductType        []string `json:"product_type"`
		Route              []string `json:"route"`
		SubstanceName      []string `json:"substance_name"`
		Rxcui              []string `json:"rxcui"`
		SplID              []string `json:"spl_id"`
		SplSetID           []string `json:"spl_set_id"`
		PackageNdc         []string `json:"package_ndc"`
		IsOriginalPackager []bool   `json:"is_original_packager"`
		Upc                []string `json:"upc"`
		Nui                []string `json:"nui"`
		PharmClassEpc      []string `json:"pharm_class_epc"`
		PharmClassMoa      []string `json:"pharm_class_moa"`
		Unii               []string `json:"unii"`
	} `json:"openfda"`
}

type fdaLabelData struct {
	Meta struct {
		Disclaimer  string `json:"disclaimer"`
		Terms       string `json:"terms"`
		License     string `json:"license"`
		LastUpdated string `json:"last_updated"`
		Results     struct {
			Skip  int `json:"skip"`
			Limit int `json:"limit"`
			Total int `json:"total"`
		} `json:"results"`
	} `json:"meta"`
	Results []fdaLabelResult `json:"results"`
}

const fdaLabelAPIBase = "https://api.fda.gov/drug/label.json" // ?search=<brand_name>
const rateLimitSeconds = 2

// fdaLabelRecencyLookup looks up the most recent FDA label information for a given brand name.
// if the label has been updated since lastChecked, it returns the new effective date.
func fdaLabelRecencyLookup(brandNames []string) (map[string]time.Time, error) {
	fmt.Println("starting FDA label recency lookup for", len(brandNames), "brand names")
	fmt.Printf("network will take %d sec for rate limiting.\n", rateLimitSeconds*len(brandNames))
	l := rate.NewLimiter(rate.Every(rateLimitSeconds*time.Second), 1)
	results := make(map[string]time.Time)
	for _, brandName := range brandNames {
		fmt.Print("Checking FDA label for brand name:", brandName, "...")
		if err := l.Wait(context.Background()); err != nil {
			return nil, errors.Join(errors.New("error waiting for rate limiter"), err)
		}
		u, _ := url.Parse(fdaLabelAPIBase)
		q := u.Query()
		q.Set("search", brandName)
		q.Set("limit", "30")
		u.RawQuery = q.Encode()

		c := http.Client{}
		req, _ := http.NewRequest("GET", u.String(), nil)
		req.Header.Set("User-Agent", "pugnare.health/1.0")

		//fmt.Println("Making FDA API request for brand name:", brandName, "URL:", u.String())
		resp, err := c.Do(req)
		if err != nil {
			return nil, errors.Join(errors.New("error making FDA API request"), err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("FDA API returned non-200 status (%d) url: %s", resp.StatusCode, u.String())
		}
		fmt.Print(" status " + resp.Status + "...")
		var fdaLabel fdaLabelData
		if err = json.NewDecoder(resp.Body).Decode(&fdaLabel); err != nil {
			return nil, errors.Join(errors.New("failed to decode api json response"), err)
		}

		if err := resp.Body.Close(); err != nil {
			return nil, errors.Join(errors.New("error closing FDA API response body"), err)
		}

		if len(fdaLabel.Results) == 0 {
			return nil, errors.New("no FDA label results found for brand name: " + brandName + " URL: " + u.String())
		}

		// if there's more than one result, return an error
		lastChecked := time.Time{}
		for _, result := range fdaLabel.Results {
			if len(result.SplProductDataElements) == 0 {
				fmt.Println("Skipping FDA label result with empty spl_product_data_elements for brand name:", brandName, "URL:", u.String())
				continue
			}
			dataElementsFirstWord := strings.Split(result.SplProductDataElements[0], " ")[0]
			drugNameLower := strings.ToLower(dataElementsFirstWord)
			brandNameLower := strings.ToLower(brandName)
			if drugNameLower != brandNameLower {
				// fmt.Println("Not a match: drugname ", drugNameLower, "vs brandname", brandNameLower)
				continue
			}
			effectiveTime, err := time.Parse("20060102", result.EffectiveTime)
			if err != nil {
				return nil, errors.Join(errors.New("error parsing effective time from FDA label"), err)
			}
			if effectiveTime.After(lastChecked) {
				lastChecked = effectiveTime
			}
		}
		if lastChecked.IsZero() {
			return nil, errors.New("no brand-matching FDA label found for brand name: " +
				brandName + " URL: " + u.String())
		}
		results[brandName] = lastChecked
		fmt.Println(" done.")
	}
	return results, nil
}

func checkForLabelUpdates(products []product) error {
	brandNames := []string{}
	for _, p := range products {
		if p.AdminRoute == "Automatic Applicator" || p.AdminRoute == "Tubeless Insulin Pump" {
			fmt.Println("Skipping label update for:", p.BrandName)
			continue // skip CGMs, pumps, etc without FDA labels
		}
		brandNames = append(brandNames, p.BrandName)
	}

	recencyResults, err := fdaLabelRecencyLookup(brandNames)
	if err != nil {
		return errors.Join(errors.New("error looking up FDA label recency"), err)
	}

	// print out the results
	for i, p := range products {
		if !slices.Contains(brandNames, p.BrandName) {
			continue // skip products we didn't check
		}
		recency, ok := recencyResults[p.BrandName]
		if !ok {
			return fmt.Errorf("no FDA label recency found for brand name: %s", p.BrandName)
		}
		lastUpdated, err := time.Parse("2006-01-02", p.FDALabelUpdated)
		if err != nil {
			return errors.Join(fmt.Errorf("error parsing existing FDA label updated date for %s: %v", p.BrandName, err), err)
		}
		if recency.After(lastUpdated) {
			/*
				fmt.Printf("FDA label for %s has been updated since last recorded date. New effective date: %s (was %s)\n",
					p.BrandName, recency.Format("2006-01-02"), lastUpdated.Format("2006-01-02"))
			*/
			products[i].FDALabelNeedsUpdate = true
		}
	}
	return nil
}
