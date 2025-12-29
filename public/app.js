// Drug data - In your Go app, you can load this from a JSON endpoint or embed it in the template
const drugsData = [    
    {
        generic: "Tirzepatide",
        brand: "Mounjaro",
        type: "GLP-1/GIP Dual Agonist",
        route: "Subcutaneous Injection",
        dosing: "Once Weekly",
        savings: "Savings Card ($25/mo), Lilly Cares Patient Assistance Program",
        phone: "833-807-6576",
        link: "https://mounjaro.lilly.com/savings-resources",
        colorClass: "gradient-orange"
    },
    {
        generic: "Tirzepatide",
        brand: "Zepbound",
        type: "GLP-1/GIP Dual Agonist",
        route: "Subcutaneous Injection",
        dosing: "Once Weekly",
        savings: "Savings Card ($25/mo), LillyDirect ($299-449/mo)",
        phone: "1-866-923-1953",
        link: "https://zepbound.lilly.com/coverage-savings",
        colorClass: "gradient-amber"
    },
    {
        generic: "Semaglutide",
        brand: "Ozempic",
        type: "GLP-1 Agonist",
        route: "Subcutaneous Injection",
        dosing: "Once Weekly",
        savings: "Savings Card ($25/mo), Patient Assistance Program (uninsured), Medicare Prescription Payment Plan",
        phone: "1-866-310-7549",
        link: "https://www.novocare.com/diabetes/products/ozempic/savings-offer.html",
        colorClass: "gradient-blue"
    },
    {
        generic: "Semaglutide",
        brand: "Rybelsus",
        type: "GLP-1 Agonist",
        route: "Oral Tablet",
        dosing: "Once Daily",
        savings: "Savings Card ($10/mo), Patient Assistance Program (free-$80/mo)",
        phone: "1-833-275-2233",
        link: "https://www.novocare.com/diabetes/products/rybelsus/savings-offer.html",
        colorClass: "gradient-indigo"
    },
    {
        generic: "Semaglutide",
        brand: "Wegovy",
        type: "GLP-1 Agonist",
        route: "Subcutaneous Injection",
        dosing: "Once Weekly",
        savings: "Savings Card ($25/mo), Patient Assistance Program, Medicare Prescription Payment Plan",
        phone: "1-888-793-1218",
        link: "https://www.wegovy.com/coverage-and-savings/save-on-wegovy.html",
        colorClass: "gradient-pink"
    },
    {
        generic: "Liraglutide",
        brand: "Victoza",
        type: "GLP-1 Agonist",
        route: "Subcutaneous Injection",
        dosing: "Once Daily",
        savings: "Patient Assistance Program, Discount programs available",
        phone: "1-866-310-7549",
        link: "https://www.novocare.com/diabetes/products/victoza.html",
        colorClass: "gradient-teal"
    },
    {
        generic: "Dulaglutide",
        brand: "Trulicity",
        type: "GLP-1 Agonist",
        route: "Subcutaneous Injection",
        dosing: "Once Weekly",
        savings: "Savings Card ($25/mo, max $150), Lilly Cares Patient Assistance Program",
        phone: "1-844-878-4636",
        link: "https://trulicity.lilly.com/savings-resources",
        colorClass: "gradient-emerald"
    },
];

// State
let currentSearchTerm = '';
let filteredDrugs = [...drugsData];

// Icons as SVG strings
const icons = {
    pill: `<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <rect x="8" y="4" width="8" height="16" rx="4"/>
        <line x1="8" y1="12" x2="16" y2="12"/>
    </svg>`,
    externalLink: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
        <polyline points="15 3 21 3 21 9"/>
        <line x1="10" y1="14" x2="21" y2="3"/>
    </svg>`,
    phone: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z"/>
    </svg>`,
    injection: `<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="m18 2 4 4"/>
        <path d="m17 7 3-3"/>
        <path d="M19 9 8.7 19.3c-1 1-2.5 1-3.4 0l-.6-.6c-1-1-1-2.5 0-3.4L15 5"/>
        <path d="m9 11 4 4"/>
        <path d="m5 19-3 3"/>
        <path d="m14 4 6 6"/>
    </svg>`
};

// Create drug card HTML
function createDrugCard(drug, index) {
    return `
        <div class="drug-card" style="animation-delay: ${0.5 + index * 0.05}s">
            <div class="drug-card-accent ${drug.colorClass}"></div>
            <div class="drug-card-content">
                <div class="drug-card-inner">
                    <div class="drug-info">
                        <div class="drug-header">
                            <div class="drug-icon ${drug.colorClass}">
                                ${((drug.route == "Oral Tablet") ? icons.pill : icons.injection)}
                            </div>
                            <div>
                                <h3 class="drug-name">${drug.brand}</h3>
                                <p class="drug-subtitle">${drug.generic} • ${drug.type}</p>
                            </div>
                        </div>

                        <div class="drug-details">
                            <div class="drug-detail">
                                <p class="drug-detail-label">Administration</p>
                                <p class="drug-detail-value">${drug.route}</p>
                            </div>
                            <div class="drug-detail">
                                <p class="drug-detail-label">Dosing</p>
                                <p class="drug-detail-value">${drug.dosing}</p>
                            </div>
                        </div>

                        <div class="drug-savings">
                            <p class="drug-savings-label">💰 Available Savings Programs</p>
                            <p class="drug-savings-programs">${drug.savings}</p>
                        </div>
                    </div>

                    <div class="drug-actions">
                        <a href="${drug.link}" 
                           target="_blank" 
                           rel="noopener noreferrer" 
                           class="btn btn-primary ${drug.colorClass}">
                            <span>View Savings Programs</span>
                            ${icons.externalLink}
                        </a>
                        <a href="tel:${drug.phone}" class="btn btn-secondary">
                            ${icons.phone}
                            <span>${drug.phone}</span>
                        </a>
                    </div>
                </div>
            </div>
        </div>
    `;
}

// Render drug cards
function renderDrugCards(drugs) {
    const container = document.getElementById('drug-cards-container');
    const noResults = document.getElementById('no-results');

    if (drugs.length === 0) {
        container.innerHTML = '';
        noResults.style.display = 'block';
    } else {
        noResults.style.display = 'none';
        container.innerHTML = drugs.map((drug, index) => createDrugCard(drug, index)).join('');
    }
}

/*
// Filter drugs based on search term
function filterDrugs(searchTerm) {
    const term = searchTerm.toLowerCase().trim();
    
    if (!term) {
        return [...drugsData];
    }

    return drugsData.filter(drug => 
        drug.brand.toLowerCase().includes(term) ||
        drug.generic.toLowerCase().includes(term) ||
        drug.type.toLowerCase().includes(term)
    );
}

// Handle search input
function handleSearch(event) {
    currentSearchTerm = event.target.value;
    filteredDrugs = filterDrugs(currentSearchTerm);
    renderDrugCards(filteredDrugs);
}
*/

// Update drug count in header
function updateDrugCount() {
    const countElement = document.getElementById('drug-count');
    if (countElement) {
        countElement.textContent = `${drugsData.length} medications • Multiple savings options`;
    }
}

// Initialize the app
function init() {
    // Set up search input listener
    const searchInput = document.getElementById('search-input');
    if (searchInput) {
        searchInput.addEventListener('input', handleSearch);
    }

    // Update drug count
    updateDrugCount();

    // Initial render
    renderDrugCards(filteredDrugs);
}

// Initialize when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
} else {
    init();
}

// Export for potential use in Go templates
// You can access drugsData if you need to inject it server-side
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { drugsData };
}