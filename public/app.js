let filteredDrugs = [...drugsData];

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