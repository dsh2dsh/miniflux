class BoostedBody {
  constructor() {
    const body = document.body;
    body.addEventListener("htmx:before:history:restore",
      event => this.historyCache(event));
    body.addEventListener("htmx:after:request",
      event => this.afterRequest(event));
    body.addEventListener("htmx:before:swap", event => this.beforeSwap(event));
    body.addEventListener("htmx:after:settle", event => this.afterSwap(event));
  }

  historyCache(event) {
    readOnScrollObserver.stop();
  }

  afterRequest(event) {
    if (!this.boosted(event)) return;
    event.detail.ctx.swap = "innerHTML show:top showTarget:html"
  }

  beforeSwap(event) {
    if (!this.boosted(event)) return;
    readOnScrollObserver.stop();
  }

  afterSwap(event) {
    if (!this.boosted(event)) return;

    initializeFormHandlers();
    initializeMediaPlayerHandlers();
    readOnScrollObserver.addEntries();
  }

  boosted(event) {
    if (event.detail.ctx)
      return event.detail.ctx.target === document.body;
    return event.target === document.body;
  }
}

const boostedBody = new BoostedBody();
