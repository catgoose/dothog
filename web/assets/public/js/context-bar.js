// setup:feature:demo
/**
 * Alpine.js component that populates the context bar from <link> tags.
 * Reads bookmark (frecent) and related link relations from the document head.
 * @returns {AlpineComponent}
 */
function contextBar() {
  return {
    frecent: [],
    related: [],
    init() {
      this.frecent = this.readLinks('bookmark');
      this.related = this.readLinks('related');
    },
    readLinks(rel) {
      return Array.from(document.querySelectorAll('head link[rel="' + rel + '"]'))
        .map(function(el) { return { href: el.getAttribute('href'), title: el.getAttribute('title') }; })
        .filter(function(l) { return l.href && l.title && l.href !== window.location.pathname; });
    }
  };
}
