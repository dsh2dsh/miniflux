async function messageConfirmed(url, redirectURL) {
  try {
    const resp = await fetch(url, {
      method: "POST",
      headers: {
        "Accept": "application/json",
      },
      redirect: "error",
    });
    if (!resp.ok)
      throw new Error(`Response status: ${resp.status}`);

    if (redirectURL) {
      location.replace(redirectURL);
      return;
    }

    const result = await resp.json();
    location.replace(result.url || location.href);
  } catch (error) {
    console.error(error.message);
  }
}

async function markItemsRead(items) {
  const unreadItems = items.filter((el) => el.matches(".item-status-unread"));
  if (!unreadItems.length) return;

  const entryIDs = unreadItems.map((el) => parseInt(el.dataset.id, 10));
  const statusRead = "read";

  await new Promise((resolve) => {
    updateEntriesStatus(entryIDs, statusRead, () => resolve());
  });

  unreadItems.forEach((el) => {
    const toggleStatus = el.querySelector(":is(a, button)[data-toggle-status]");
    if (toggleStatus) {
      setReadStatusButtonState(toggleStatus, statusRead);
    }
    el.classList.replace("item-status-unread", "item-status-read");
  });
}

function initDataConfirm() {
  document.body.addEventListener("click", (event) => {
    if (event.target.closest(":is(a, button)[data-confirm]")) {
      handleConfirmationMessage(event.target, messageConfirmed);
    };
  });
}

function createScriptURL(src) {
  const ttpolicy = trustedTypes
    .createPolicy('url', {createScriptURL: src => src});
  return ttpolicy.createScriptURL(src);
}

function initServiceWorker() {
  if ("serviceWorker" in navigator === false) return;

  const serviceWorkerURL = document.body.dataset.serviceWorkerUrl;
  if (!serviceWorkerURL) return;

	navigator.serviceWorker.
    register(createScriptURL(serviceWorkerURL), {type: "module"}).
    catch((error) => {
      console.warn(`Service worker registration failed: ${error}`);
    });
}

function initCommentLinks() {
  document.body.addEventListener("click", (event) => {
    if (event.target.closest("a[data-comments-link=true]")) {
      handleEntryStatus("next", event.target, true);
    }
  });
}

initDataConfirm();
initServiceWorker();
initCommentLinks();
