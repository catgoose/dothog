/**
 * Interval control helpers called from hyperscript.
 * Each interval slider wrapper <div> stores state in data-* attributes
 * and a _ivUnit expando property for the current unit index.
 */
(function () {
  var units = ['ms', 's', 'min', 'h'];
  var configs = {
    ms:  { min: 100, max: 2000, step: 100, mult: 1 },
    s:   { min: 1,   max: 60,   step: 1,   mult: 1000 },
    min: { min: 1,   max: 60,   step: 1,   mult: 60000 },
    h:   { min: 1,   max: 24,   step: 1,   mult: 3600000 }
  };

  /** Called once via hyperscript init to set the unit index. */
  window._ivInit = function (el) {
    var unit = el.dataset.unit || 's';
    var idx = units.indexOf(unit);
    el._ivUnit = idx >= 0 ? idx : 1;
  };

  /** Cycle unit: ms → s → min → h → ms. Reconfigure slider and POST. */
  window._ivCycle = function (el) {
    var input = el.querySelector('input[type=range]');
    var display = el.querySelector('span');
    var btn = el.querySelector('button');
    if (!input) return;

    var oldCfg = configs[units[el._ivUnit]];
    var ms = parseInt(input.value) * oldCfg.mult;

    el._ivUnit = (el._ivUnit + 1) % units.length;
    var unit = units[el._ivUnit];
    var cfg = configs[unit];

    var val = Math.round(ms / cfg.mult);
    if (val < cfg.min) val = cfg.min;
    if (val > cfg.max) val = cfg.max;

    input.min = cfg.min;
    input.max = cfg.max;
    input.step = cfg.step;
    input.value = val;
    if (display) display.textContent = val;
    if (btn) btn.textContent = unit;

    _ivPost(el);
  };

  /** POST the current interval in ms. */
  window._ivPost = function (el) {
    var input = el.querySelector('input[type=range]');
    if (!input) return;
    var cfg = configs[units[el._ivUnit || 0]];
    var ms = parseInt(input.value) * cfg.mult;

    var url = el.dataset.postUrl;
    var key = el.dataset.targetKey;
    var value = el.dataset.targetValue;
    if (!url) return;

    var params = new URLSearchParams();
    params.set(key, value);
    params.set('interval_ms', ms.toString());
    var t = document.querySelector('meta[name="csrf-token"]');
    fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
        'X-CSRF-Token': t ? t.content : ''
      },
      body: params.toString()
    });
  };
})();
