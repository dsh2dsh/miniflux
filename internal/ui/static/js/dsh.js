async function messageConfirmed(url, redirectURL) {
  try {
    const resp = await fetch(url, {
      method: "POST",
      headers: {
        "X-Requested-With": "messageConfirmed",
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

function initDataConfirm() {
  document.body.addEventListener("click", (event) => {
    if (event.target.closest(":is(a, button)[data-confirm]")) {
      handleConfirmationMessage(event.target, messageConfirmed);
    };
  });
}

function initServiceWorker() {
  if ("serviceWorker" in navigator === false) return;

  const serviceWorkerURL = document.body.dataset.serviceWorkerUrl;
  if (!serviceWorkerURL) return;

	navigator.serviceWorker.
    register(ttpolicy.createScriptURL(serviceWorkerURL), {type: "module"}).
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
