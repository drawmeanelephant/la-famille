(function() {
    document.addEventListener("DOMContentLoaded", () => {
        const searchInput = document.getElementById("site-search");
        const resultsContainer = document.getElementById("search-results-list");

        if (!searchInput) return; // Search not available on this page

        // Global keydown listener for `/` and `Escape`
        document.addEventListener("keydown", (e) => {
            if (e.key === "/" && !["INPUT", "TEXTAREA", "SELECT"].includes(document.activeElement.tagName) && !document.activeElement.isContentEditable) {
                e.preventDefault();
                searchInput.focus();
            } else if (e.key === "Escape" && resultsContainer && !resultsContainer.classList.contains("hidden")) {
                resultsContainer.classList.add("hidden");
                searchInput.blur();
            }
        });

        let fetchPromise = null;

        // Under a siteurl subpath (e.g. GitHub Pages project sites), the rendered
        // page carries a base path on every root-relative URL; the search index
        // lives under that same base, not at the domain root (#528).
        const searchBase = (() => {
            const el = document.querySelector('meta[name="la-famille-base-path"]');
            return el ? (el.getAttribute("content") || "") : "";
        })();

        const fetchMetaData = () => {
            if (!fetchPromise) {
                fetchPromise = fetch(searchBase + "/search.json")
                    .then(response => {
                        if (!response.ok) throw new Error("Network response was not ok");
                        return response.json();
                    })
                    .then(data => {
                        window.LaFamilleSearchIndex = data;
                        return data;
                    })
                    .catch(e => {
                        console.error("Failed to fetch search index:", e);
                        fetchPromise = null;
                    });
            }
            return fetchPromise;
        };

        // Attach `{ once: true }` focus listener
        searchInput.addEventListener("focus", fetchMetaData, { once: true });

        let debounceTimeout;

        searchInput.addEventListener("input", (e) => {
            clearTimeout(debounceTimeout);
            debounceTimeout = setTimeout(async () => {
                const query = e.target.value.toLowerCase().trim();

                if (!resultsContainer) return; // Search results container not found

                if (!query) {
                    resultsContainer.classList.add("hidden");
                    resultsContainer.innerHTML = "";
                    return;
                }

                await fetchMetaData();

                if (!window.LaFamilleSearchIndex) {
                    resultsContainer.classList.add("hidden");
                    resultsContainer.innerHTML = "";
                    return;
                }

                const results = window.LaFamilleSearchIndex.filter(item => {
                    const titleMatch = (item.t || "").toLowerCase().includes(query);
                    const tagMatch = (item.g || []).some(tag => tag.toLowerCase().includes(query));
                    const snippetMatch = (item.s || "").toLowerCase().includes(query);
                    const headingMatch = (item.h || []).some(h => h.toLowerCase().includes(query));
                    return titleMatch || tagMatch || snippetMatch || headingMatch;
                }).slice(0, 7);

                resultsContainer.innerHTML = "";
                if (results.length === 0) {
                    const li = document.createElement("li");
                    li.className = "search-no-results";
                    li.textContent = "No results found";
                    resultsContainer.appendChild(li);
                    resultsContainer.classList.remove("hidden");
                    return;
                }

                results.forEach(item => {
                    const title = item.t || "Untitled";
                    const snippet = item.s || "";
                    const headings = item.h || [];
                    const tags = item.g || [];

                    const li = document.createElement("li");
                    const a = document.createElement("a");
                    // encodeURI on interpolated variables to prevent DOM-based XSS
                    a.href = encodeURI(item.u);
                    a.className = "search-result-link";

                    const titleDiv = document.createElement("div");
                    titleDiv.className = "search-result-title";
                    titleDiv.textContent = title;
                    a.appendChild(titleDiv);

                    const matchedHeading = headings.find(h => h.toLowerCase().includes(query));
                    if (matchedHeading) {
                        const headingDiv = document.createElement("div");
                        headingDiv.className = "search-result-section";
                        headingDiv.textContent = "Section: " + matchedHeading;
                        a.appendChild(headingDiv);
                    }

                    if (snippet) {
                        const snippetDiv = document.createElement("div");
                        snippetDiv.className = "search-result-snippet";
                        snippetDiv.textContent = snippet;
                        a.appendChild(snippetDiv);
                    }

                    if (tags.length > 0) {
                        const tagsDiv = document.createElement("div");
                        tagsDiv.className = "search-result-tags";
                        tags.forEach(tag => {
                            const badge = document.createElement("span");
                            badge.className = "search-result-tag";
                            badge.textContent = "#" + tag;
                            tagsDiv.appendChild(badge);
                        });
                        a.appendChild(tagsDiv);
                    }

                    li.appendChild(a);
                    resultsContainer.appendChild(li);
                });

                resultsContainer.classList.remove("hidden");
            }, 50); // 50ms debounce
        });

        document.addEventListener("click", (e) => {
            if (resultsContainer && !searchInput.contains(e.target) && !resultsContainer.contains(e.target)) {
                resultsContainer.classList.add("hidden");
            }
        });
    });
})();
