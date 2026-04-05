/**
 * @fileoverview HTMX swap extension that uses Alpine.morph instead of innerHTML
 * for DOM updates. Alpine.morph preserves Alpine component state (x-data,
 * watchers, effects) during swaps, which innerHTML destroys. Use
 * hx-swap="morph" on elements to opt in.
 */
htmx.defineExtension('alpine-morph', {
  /**
   * Declare that the 'morph' swap style is handled inline by this extension.
   * @param {string} swapStyle - The hx-swap value.
   * @returns {boolean} True if this extension handles the swap.
   */
  isInlineSwap: function(swapStyle) {
    return swapStyle === 'morph'
  },
  /**
   * Perform the morph swap using Alpine.morph.
   * @param {string} swapStyle - The hx-swap value.
   * @param {Element} target - The element being swapped.
   * @param {DocumentFragment|Element} fragment - The server response fragment.
   * @returns {Array<Element>|undefined} The swapped elements, or undefined if
   *   the swap style is not 'morph'.
   */
  handleSwap: function(swapStyle, target, fragment) {
    if (swapStyle === 'morph') {
      if (fragment.nodeType === Node.DOCUMENT_FRAGMENT_NODE) {
        Alpine.morph(target, fragment.firstElementChild)
        return [target]
      } else {
        Alpine.morph(target, fragment.outerHTML)
        return [target]
      }
    }
  }
})
