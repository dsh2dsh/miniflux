class MarkReadOnScroll {
  static pageEndSelector = ".items > .item.pageEnd";

  lastAdded;
  firstScrolled;
  lastScrolled;
  timeoutId = 0;

  constructor(entries) {
    document.body.addEventListener("htmx:afterSwap", event => {
      if (event.target.matches(MarkReadOnScroll.pageEndSelector))
        this.revealedCallback(event.target);
    }, true);

    history.scrollRestoration = "manual";
    this.observer = new IntersectionObserver((entries, observer) => {
      this.observerCallback(entries, observer);
    }, {
      root: null,
    });
    this.addEntries(entries);
  }

  addEntries(entries) {
    entries.forEach(entry => this.addEntry(entry));
  }

  addEntry(entry) {
    this.observer.observe(entry);
    this.lastAdded = entry;
  }

  observerCallback(entries, observer) {
    let scrolledEntry = false;
    entries.forEach((entry) => {
      const bottom = entry.boundingClientRect.bottom;
      if (entry.isIntersecting || bottom > 0)
        return;

      const element = entry.target;
      const entryId = element.dataset.id;
      this.observer.unobserve(element);

      if (!this.firstScrolled )
        this.firstScrolled = this.lastScrolled = element;
      else
        this.lastScrolled = element;
      scrolledEntry = true;
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

  revealedCallback(lastItem) {
    this.removeDupsAfter(lastItem);
    this.removeScrolled();
  }

  removeDupsAfter(lastItem) {
    if (!lastItem.nextElementSibling) return;

    const knownItems = new Set();
    let el = lastItem
    while (el) {
      knownItems.add(el.dataset.id);
      el = el.previousElementSibling;
    }

    el = lastItem.nextElementSibling;
    while (el) {
      const nextSibling = el.nextElementSibling;
      if (knownItems.has(el.dataset.id))
        el.remove();
      else
        this.addEntry(el);
      el = nextSibling;
    }
  }

  removeScrolled() {
    const pageEnds = document.querySelectorAll(MarkReadOnScroll.pageEndSelector);
    if (pageEnds.length > 2) {
      let el = pageEnds.at(-3);
      while (el) {
        const previousSibling = el.previousElementSibling;
        el.remove();
        el = previousSibling;
      }
    }
  }

  reset() {
    if (this.timeoutId > 0) {
      clearTimeout(this.timeoutId);
      this.timeoutId = 0;
    }
    this.observer.disconnect();
    this.lastAdded = this.firstScrolled = this.lastScrolled = undefined;
  }
}

function infiniteScrollEntries() {
  return document.querySelectorAll(
    ".items[data-infinite-scroll=true] > .item.item-status-unread")
}

const readOnScrollObserver = new MarkReadOnScroll(infiniteScrollEntries());
