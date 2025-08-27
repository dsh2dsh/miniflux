class MarkReadOnScroll {
  scrolledEntries = [];
  timeoutId = 0;
  lastEntryID = "";

  constructor(entries) {
    if (entries.length == 0) {
      return;
    }
    history.scrollRestoration = "manual";

    this.observer = new IntersectionObserver((entries, observer) => {
      this.observerCallback(entries, observer);
    }, {
      root: null,
    });

    entries.forEach((entry) => {
      this.observer.observe(entry);
      this.lastEntryID = entry.dataset.id;
    });
  }

  observerCallback(entries, observer) {
    let addedEntries = false;
    let pageEnd = false;

    entries.forEach((entry) => {
      const bottom = entry.boundingClientRect.bottom;
      if (entry.isIntersecting || bottom > 0) {
        return;
      }

      const element = entry.target;
      this.observer.unobserve(element);

      if (element.dataset.id === this.lastEntryID) {
        pageEnd = true;
      }

      if (!element.classList.contains("item-status-unread")) {
        return;
      }
      this.scrolledEntries.push(element);
      addedEntries = true;
    });


    if (pageEnd) {
      if (this.timeoutId > 0) {
        clearTimeout(this.timeoutId)
        this.timeoutId = 0;
      }
      this.markReadOnTimeout().finally(() => location.replace(location.href));
      return;
    }

    if (!addedEntries || this.timeoutId > 0) {
      return;
    }

    this.timeoutId = setTimeout(() => {
      this.markReadOnTimeout();
      this.timeoutId = 0;
    }, 1000);
  }

  async markReadOnTimeout() {
    const items = this.scrolledEntries.slice();
    this.scrolledEntries.length = 0;
    await markItemsRead(items);
  }
}

const readOnScrollObserver = new MarkReadOnScroll(
  document.querySelectorAll(".items .item.item-status-unread"));
