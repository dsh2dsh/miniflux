class MarkReadOnScroll {
  scrolledEntries = [];
  timeoutId = 0;

  constructor(entries) {
    if (entries.length == 0) {
      return;
    }

    this.observer = new IntersectionObserver((entries, observer) => {
      this.observerCallback(entries, observer);
    }, {
      root: null,
    });

    entries.forEach((entry) => {
      this.observer.observe(entry);
    });
  }

  observerCallback(entries, observer) {
    let addedEntries = false;
    entries.forEach((entry) => {
      const bottom = entry.boundingClientRect.bottom;
      const visible = entry.isIntersecting;
      if (visible || bottom > 0) {
        return;
      }

      const element = entry.target;
      this.observer.unobserve(element);

      if (!element.classList.contains("item-status-unread")) {
        return;
      }
      this.scrolledEntries.push(element);
      addedEntries = true;
    });

    if (!addedEntries || this.timeoutId > 0) {
      return;
    }

    this.timeoutId = setTimeout(() => {
      this.markReadOnTimeout();
      this.timeoutId = 0;
    }, 1000);
  }

  markReadOnTimeout() {
    const items = this.scrolledEntries.slice();
    this.scrolledEntries.length = 0;

    const entryIDs = items.map((element) => parseInt(element.dataset.id, 10));
    updateEntriesStatus(entryIDs, "read", () => {
      items.forEach((element) => {
        element.classList.replace("item-status-unread", "item-status-read");
      });
    });
  }
}

let readOnScrollObserver;
function markReadOnScroll() {
  const entries = document.querySelectorAll(
    '.items[data-mark-read-on-scroll="1"] .item.item-status-unread');
  if (entries.length == 0) {
    return;
  }
  readOnScrollObserver = new MarkReadOnScroll(entries);

  const selector = 'a[data-page="next"]';
  const nextPage = document.querySelector(selector);
  if (!nextPage) {
    return;
  }

  const lastSelector = 'a[data-page="last"]';
  const lastPage = document.querySelector(lastSelector);
  if (lastPage) {
    const nextOffset = parseInt(nextPage.dataset.offset, 10);
    const lastOffset = parseInt(lastPage.dataset.offset, 10);
    if (lastOffset == nextOffset) {
      selector.concat(", ", lastSelector);
    }
  }

  document.querySelectorAll(selector).forEach((element) => {
    element.addEventListener("click", (event) => {
      event.preventDefault();
      markPageAsRead();
    });
  });
}
