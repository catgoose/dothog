// setup:feature:demo
/**
 * Alpine.js component that populates the context bar from <link rel="related"> tags.
 * Reads link relations from the document head, groups them by data-group attribute,
 * and renders them as navigation links with group labels.
 * @returns {AlpineComponent}
 */
function contextBar() {
  return {
    groups: [],
    init() {
      var links = Array.from(document.querySelectorAll('head link[rel="related"]'))
        .map(function(el) {
          return {
            href: el.getAttribute('href'),
            title: el.getAttribute('title'),
            group: el.getAttribute('data-group') || ''
          };
        })
        .filter(function(l) { return l.href && l.title && l.href !== window.location.pathname; });

      // Group links by their group name, preserving order
      var seen = {};
      var groups = [];
      for (var i = 0; i < links.length; i++) {
        var key = links[i].group;
        if (!seen[key]) {
          seen[key] = { name: key, links: [] };
          groups.push(seen[key]);
        }
        seen[key].links.push(links[i]);
      }
      this.groups = groups;
    },
    hasLinks() {
      return this.groups.length > 0;
    }
  };
}
