class BoostedBody {
  constructor() {
    const body = document.body;
    body.addEventListener("htmx:beforeSwap", event => this.beforeSwap(event));
    body.addEventListener("htmx:afterSwap", event => this.afterSwap(event));
  }

  beforeSwap(event) {
    if (!this.boosted(event)) return;
    event.detail.swapOverride = "show:html:top";
  }

  afterSwap(event) {
    if (!this.boosted(event)) return;

    initializeFormHandlers();
    initializeMediaPlayerHandlers();
    readOnScrollObserver.restart();
  }

  boosted(event) {
    return event.target === document.body && event.detail.boosted;
  }
}

const boostedBody = new BoostedBody();
