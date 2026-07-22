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

        const fetchMetaData = () => {
            if (!fetchPromise) {
                fetchPromise = fetch("/search.json")
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
                    li.className = "p-2 text-base-content/50";
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
                    a.className = "block p-4 hover:bg-base-200 text-sm focus-visible:bg-base-200 focus-visible:outline-none border-b border-base-200 last:border-0";

                    const titleDiv = document.createElement("div");
                    titleDiv.className = "font-bold text-base-content";
                    titleDiv.textContent = title;
                    a.appendChild(titleDiv);

                    const matchedHeading = headings.find(h => h.toLowerCase().includes(query));
                    if (matchedHeading) {
                        const headingDiv = document.createElement("div");
                        headingDiv.className = "text-xs font-medium text-primary mt-0.5";
                        headingDiv.textContent = "Section: " + matchedHeading;
                        a.appendChild(headingDiv);
                    }

                    if (snippet) {
                        const snippetDiv = document.createElement("div");
                        snippetDiv.className = "text-xs text-base-content/70 mt-1 line-clamp-2";
                        snippetDiv.textContent = snippet;
                        a.appendChild(snippetDiv);
                    }

                    if (tags.length > 0) {
                        const tagsDiv = document.createElement("div");
                        tagsDiv.className = "flex flex-wrap gap-1 mt-1.5";
                        tags.forEach(tag => {
                            const badge = document.createElement("span");
                            badge.className = "badge badge-xs badge-ghost text-[10px]";
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
