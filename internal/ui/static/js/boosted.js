class BoostedBody {
  constructor() {
    const body = document.body;
    body.addEventListener("htmx:historyCacheMissLoad",
      event => this.historyCache(event));
    body.addEventListener("htmx:beforeSwap", event => this.beforeSwap(event));
    body.addEventListener("htmx:afterSwap", event => this.afterSwap(event));
  }

  historyCache(event) {
    readOnScrollObserver.stop();
  }

  beforeSwap(event) {
    if (!this.boosted(event)) return;
    event.detail.swapOverride = "show:html:top";
    readOnScrollObserver.stop();
  }

  afterSwap(event) {
    if (!this.boosted(event)) return;

    initializeFormHandlers();
    initializeMediaPlayerHandlers();
    readOnScrollObserver.addEntries();
  }

  boosted(event) {
    return event.target === document.body && event.detail.boosted;
  }
}

const boostedBody = new BoostedBody();
