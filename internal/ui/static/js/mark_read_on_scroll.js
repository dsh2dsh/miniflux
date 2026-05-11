class MarkReadOnScroll {
  static pageEndSelector = ".items > .item.pageEnd";

  lastAdded;
  firstScrolled;
  lastScrolled;
  timeoutId = 0;

  constructor() {
    document.body.addEventListener("htmx:afterSwap", event => {
      if (event.target.matches(MarkReadOnScroll.pageEndSelector))
        this.revealedLastItem(event.target);
    }, true);

    history.scrollRestoration = "manual";
    this.observer = new IntersectionObserver((entries, observer) => {
      this.observerCallback(entries, observer);
    }, {
      root: null,
    });
    this.addEntries();
  }

  addEntries() {
    document.querySelectorAll(
      ".items[data-infinite-scroll=true] > .item.item-status-unread"
    ).forEach(entry => this.addEntry(entry));
  }

  addEntry(entry) {
    this.observer.observe(entry);
    this.lastAdded = entry;
  }

  observerCallback(entries, observer) {
    let scrolledEntry = false;
    entries.forEach((entry) => {
      this.intersectingLastItem(entry);
      scrolledEntry = this.scrolledUp(entry);
    });
    if (!scrolledEntry) return;

    if (this.lastScrolled === this.lastAdded) {
      if (this.timeoutId > 0)
        clearTimeout(this.timeoutId);
      this.readOnTimeout().finally(() => location.replace(location.href));
      return;
    }

    if (this.timeoutId === 0)
      this.timeoutId = setTimeout(() => this.readOnTimeout(), 1000);
  }

  intersectingLastItem(entry) {
    if (!entry.isIntersecting) return;

    const el = entry.target;
    if (el.nextElementSibling || el.dataset.triggerRevealed !== "true")
      return;

    const e = new Event("miniflux:revealed");
    el.dispatchEvent(e);
  }

  scrolledUp(entry) {
    const bottom = entry.boundingClientRect.bottom;
    if (entry.isIntersecting || bottom > 0)
      return false;

    const element = entry.target;
    const entryId = element.dataset.id;
    this.observer.unobserve(element);

    if (!this.firstScrolled )
      this.firstScrolled = this.lastScrolled = element;
    else
      this.lastScrolled = element;
    return true
  }

  async readOnTimeout() {
    this.timeoutId = 0;
    await markItemsRead(this.scrolledEntries());
  }

  scrolledEntries() {
    const entries = [];
    let el = this.firstScrolled;
    while (el) {
      entries.push(el);
      if (el === this.lastScrolled )
        break;
      el = el.nextElementSibling;
    }

    this.firstScrolled = this.lastScrolled = null;
    return entries;
  }

  revealedLastItem(item) {
    this.addNextEntries(item);
    this.removeScrolled();
  }

  addNextEntries(lastItem) {
    let el = lastItem.nextElementSibling;
    while (el) {
      this.addEntry(el);
      el = el.nextElementSibling;
    }
  }

  removeScrolled() {
    const pageEnds = document.querySelectorAll(MarkReadOnScroll.pageEndSelector);
    if (pageEnds.length <= 2) return;

    let el = Array.from(pageEnds).at(-3);
    while (el) {
      const previousSibling = el.previousElementSibling;
      el.remove();
      el = previousSibling;
    }
  }

  stop() {
    if (this.timeoutId > 0) {
      clearTimeout(this.timeoutId);
      this.timeoutId = 0;
    }
    this.observer.disconnect();
    this.lastAdded = this.firstScrolled = this.lastScrolled = undefined;
  }
}

const readOnScrollObserver = new MarkReadOnScroll();
