(function() {
    // Prevent DOM XSS on user content
    function escapeHtml(unsafe) {
        if (!unsafe) return "";
        return unsafe
             .toString()
             .replace(/&/g, "&amp;")
             .replace(/</g, "&lt;")
             .replace(/>/g, "&gt;")
             .replace(/"/g, "&quot;")
             .replace(/'/g, "&#039;");
    }

    document.addEventListener("DOMContentLoaded", () => {
        const searchInput = document.getElementById("site-search");
        const resultsContainer = document.getElementById("search-results-list");

        if (!searchInput) return; // Search not available on this page

        // Global keydown listener for `/`
        document.addEventListener("keydown", (e) => {
            if (e.key === "/" && !["INPUT", "TEXTAREA", "SELECT"].includes(document.activeElement.tagName)) {
                e.preventDefault();
                searchInput.focus();
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

                if (!window.LaFamilleSearchIndex) return;

                const results = window.LaFamilleSearchIndex.filter(item => {
                    const titleMatch = (item.t || "").toLowerCase().includes(query);
                    const tagMatch = (item.g || []).some(tag => tag.toLowerCase().includes(query));
                    const snippetMatch = (item.s || "").toLowerCase().includes(query);
                    return titleMatch || tagMatch || snippetMatch;
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

                    const li = document.createElement("li");
                    const a = document.createElement("a");
                    // Important: encodeURI on interpolated variables to prevent DOM-based XSS
                    a.href = encodeURI(item.u);
                    a.className = "block p-4 hover:bg-base-200 text-sm focus-visible:bg-base-200 focus-visible:outline-none border-b border-base-200 last:border-0";

                    const titleDiv = document.createElement("div");
                    titleDiv.className = "font-bold text-base-content";
                    titleDiv.innerHTML = escapeHtml(title); // Use our custom escaper!

                    const snippetDiv = document.createElement("div");
                    snippetDiv.className = "text-xs text-base-content/70 mt-1 line-clamp-2";
                    snippetDiv.innerHTML = escapeHtml(snippet); // Use our custom escaper!

                    a.appendChild(titleDiv);
                    if (snippet) {
                        a.appendChild(snippetDiv);
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
