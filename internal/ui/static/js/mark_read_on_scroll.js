class MarkReadOnScroll {
  lastAdded;
  firstScrolled;
  lastScrolled;
  timeoutId = 0;

  constructor(entries) {
    history.scrollRestoration = "manual";
    this.observer = new IntersectionObserver((entries, observer) => {
      this.observerCallback(entries, observer);
    }, {
      root: null,
    });
    this.addEntries(entries);
  }

  addEntries(entries) {
    entries.forEach((entry) => {
      this.observer.observe(entry);
      this.lastAdded = entry;
    });
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
}

const readOnScrollObserver = new MarkReadOnScroll(
  document.querySelectorAll(".items .item.item-status-unread"));
