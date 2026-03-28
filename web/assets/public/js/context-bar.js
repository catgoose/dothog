// setup:feature:demo
/**
 * Alpine.js component for the composable context bar.
 * Reads <link> tags from <head> grouped by rel type and renders them as sections.
 * Each section has a rel type, alignment, and visual style.
 * @returns {AlpineComponent}
 */
function contextBar() {
  return {
    sections: [],
    init() {
      this.sections = [
        { rel: 'bookmark', links: this.readLinks('bookmark'), style: 'frecent' },
        { rel: 'related',  links: this.readLinks('related'),  style: 'related' },
      ].filter(function(s) { return s.links.length > 0; });
    },
    readLinks(rel) {
      return Array.from(document.querySelectorAll('head link[rel="' + rel + '"]'))
        .map(function(el) { return { href: el.getAttribute('href'), title: el.getAttribute('title') }; })
        .filter(function(l) { return l.href && l.title && l.href !== window.location.pathname; });
    }
  };
}
