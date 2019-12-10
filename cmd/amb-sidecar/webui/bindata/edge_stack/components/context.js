if (window != null && window.__do_not_use_or_probably_fired_context == null) {
  window.__do_not_use_or_probably_fired_context = {};
}

/**
 * Register a handler for when context changes.
 *
 * @param {string} name
 *  The name of the context to listen too.
 * @param {(any) => void} onChange
 *  A function to call when context changes.
 */
export function registerContextChangeHandler(name, onChange) {
  if (window.__do_not_use_or_probably_fired_context[name] == null) {
    throw "Context has not yet been created!";
  }
  window.__do_not_use_or_probably_fired_context[name].handlers.push(onChange);
}

/**
 * Create a new piece of context.
 *
 * @param {string} name
 *  The name of this context, should be globally unique.
 * @param {any} initialValue
 *  The initial value of this state.
 * @return
 *  An array with two elements: [value, updateValue]
 */
export function useContext(name, initialValue) {
  if (window.__do_not_use_or_probably_fired_context[name] != null) {
    return [window.__do_not_use_or_probably_fired_context[name].data, (newData) => {
      window.__do_not_use_or_probably_fired_context[name].data = newData;
      window.__do_not_use_or_probably_fired_context[name].handlers.forEach((handler) => {
        handler(newData);
      });
    }];
  }

  window.__do_not_use_or_probably_fired_context[name] = {
    data: initialValue,
    handlers: []
  };
  return [initialValue, (newData) => {
    window.__do_not_use_or_probably_fired_context[name].data = newData;
    window.__do_not_use_or_probably_fired_context[name].handlers.forEach((handler) => {
      handler(newData);
    });
  }];
}
