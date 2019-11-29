# UI Gotchas #

If this is your first time working with JavaScript in awhile, don't feel bad.
Here we'll describe some common gotchas when working with UI Components that
have been lifted from actual code.

This document presumes you've looked over lit-html, and have a basic understanding
component. This documents common gotchas we've seen when writing components.

## Using .requestUpdate() ##

***Importance: Critical***

Using requestUpdate is almost always a "bad idea" in LitElement. In general
what it signals to the framework is: "I want to control rendering myself".
This gets hairy when:

  1. You trigger too many updates too fast (are you doing your own debouncing, and deduping?)
  2. You trigger a rerender in the middle of another rerender (can cause browsers to stutter).
  3. State updates in general can get tricky with children because of the relationship needed
     between the two. More specifically it is possible for a child to update, trigger an
     update with `requestUpdate()`, and not have the parent re-render it.

So how do you handle this? The easiest way is to just let the framework
determine _when to render_. This is done by defining a `static get properties()`
function that returns an object of what properties to watch for modifications.
This is not _all of your properties_ this is only the ones that impact rendering.

For example, a component might look something like this:

```javascript
class MyComponent extends LitElement {
  static get properties() {
    return {
			data: { type: String },
      metadata: { affects_render: { type: String } }
    };
  }

  constructor() {
    super();

		this.data = 'data';
    this.metadata = {
      affects_render: 'World',
    };
    this.does_not_affect_render = 'blah';
  }

  onClick() {
    this.metadata = { ...this.metadata, affects_render: "blue" };
		console.log('OnClick', this.metadata);
  }

  onClickTwo() {
    this.does_not_affect_render = "newer";
		console.log('OnClickTwo', this.metadata);
  }

	onClickThree() {
		this.data = 'newer';
		console.log('OCT', this.data);
	}

  render() {
		console.log('render!', this.metadata);

		return html`
      <button @click="${this.onClick}">Trigger Re-Render</button>
      <button @click="${this.onClickTwo}">Change Prop</button>
			<button @click="${this.onClickThree}">Also Trigger Re-Render</button>
      <div>${this.metadata.affects_render}</div>
			<div>${this.data}</div>
    `;
  }
}

customElements.define('my-component', MyComponent);
```

Go ahead, and play with this locally. You'll notice that it only renders when
the two properties are updated that are in `static get properties()`. We will
not rerender on the second button being pressed.

## Depending on Array Ordering ##

***Importance: Nitpick***

A common thing to do in javascript is depend on array ordering. Most of the
time the ordering is guaranteed to be stable, however problems can arise in
some very specific edge cases. These specifically occur when comparing two
arrays with each other, or removing elements. So if you have two arrays,
and you want to compare between them, or you expect elements to be
removed than you'll want to ensure you follow one of the below.

There are a couple things you can do:

  1. Not depend on the ordering. Simply: `.forEach()` your way to victory.
  2. Use a custom sort function, and call sort before depending on ordering.
  3. Use a custom key of the objects to track potentially different index values.

## Creating Unnecessary "Wrapper" Divs ##

***Importance: Nitpick***

One common thing you may be tempted to do when building a component
is create a "wrapper" div. Something like:

```javascript
render() {
  return html`
  <div id="my_cool__component">
    <a href="#">A Link</a>
    <span>Another Span</span>
    <span>Yet Another Span</span>
  </div>
  `;
}
```

This may be for one of couple reasons:

  1. You're using it for styling.
  2. It's just more natural to write.

We'd prefer you don't do this because it gives the browser more work to do.
Those are extra nodes that the browser has to maintain, and each event is a bit
slower since it needs to bubble up through that node that isn't doing anything.

Instead we'd want to write:

```javascript
render() {
  return html`
    <a href="#">A Link</a>
    <span>Another Span</span>
    <span>Yet Another Span</span>
  `;
}
```

  1. For Styling prefer the ":host" pseduo selector in CSS.
  2. Hopefully you'll get used to the newer way.
