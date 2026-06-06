(function () {
  "use strict";

  var input = document.getElementById("doc-search");
  var resultsEl = document.getElementById("search-results");
  if (!input || !resultsEl) return;

  var fuse = null;
  var items = [];
  var activeIndex = -1;

  function indexURL() {
    var root = document.documentElement.getAttribute("data-baseurl");
    if (root) {
      return new URL("search-index.json", root).href;
    }
    return new URL("search-index.json", window.location.href).href;
  }

  function setExpanded(open) {
    input.setAttribute("aria-expanded", open ? "true" : "false");
    resultsEl.hidden = !open;
  }

  function clearResults() {
    resultsEl.innerHTML = "";
    activeIndex = -1;
    setExpanded(false);
  }

  function renderResults(matches) {
    resultsEl.innerHTML = "";
    activeIndex = -1;
    if (!matches.length) {
      setExpanded(false);
      return;
    }

    matches.forEach(function (match, i) {
      var item = match.item;
      var el = document.createElement("a");
      el.className = "search-result";
      el.href = item.url;
      el.setAttribute("role", "option");
      el.id = "search-result-" + i;

      var title = document.createElement("span");
      title.className = "search-result-title";
      title.textContent = item.title;
      el.appendChild(title);

      if (item.section) {
        var section = document.createElement("span");
        section.className = "search-result-section";
        section.textContent = item.section;
        el.appendChild(section);
      }

      resultsEl.appendChild(el);
    });

    setExpanded(true);
  }

  function highlightActive() {
    var links = resultsEl.querySelectorAll(".search-result");
    links.forEach(function (link, i) {
      link.classList.toggle("is-active", i === activeIndex);
      if (i === activeIndex) {
        input.setAttribute("aria-activedescendant", link.id);
      }
    });
    if (activeIndex < 0) {
      input.removeAttribute("aria-activedescendant");
    }
  }

  function search(query) {
    if (!fuse || !query.trim()) {
      clearResults();
      return;
    }
    renderResults(fuse.search(query, { limit: 8 }));
  }

  fetch(indexURL())
    .then(function (res) {
      if (!res.ok) throw new Error("search index not found");
      return res.json();
    })
    .then(function (data) {
      items = data;
      fuse = new Fuse(items, {
        keys: [
          { name: "title", weight: 0.5 },
          { name: "section", weight: 0.2 },
          { name: "content", weight: 0.3 },
        ],
        threshold: 0.4,
        ignoreLocation: true,
      });
    })
    .catch(function () {
      input.placeholder = "Search unavailable";
      input.disabled = true;
    });

  input.addEventListener("input", function () {
    search(input.value);
  });

  input.addEventListener("keydown", function (e) {
    var links = resultsEl.querySelectorAll(".search-result");
    if (!links.length) return;

    if (e.key === "ArrowDown") {
      e.preventDefault();
      activeIndex = Math.min(activeIndex + 1, links.length - 1);
      highlightActive();
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      activeIndex = Math.max(activeIndex - 1, 0);
      highlightActive();
    } else if (e.key === "Enter" && activeIndex >= 0) {
      e.preventDefault();
      window.location.href = links[activeIndex].href;
    } else if (e.key === "Escape") {
      clearResults();
      input.blur();
    }
  });

  document.addEventListener("click", function (e) {
    if (!e.target.closest(".search")) {
      clearResults();
    }
  });
})();
