function messageConfirmed(url, redirectURL) {
  fetch(url, {
    method: "POST",
    headers: {
      "X-Csrf-Token": document.body.dataset.csrfToken || ""
    },
    redirect: "manual",
  }).then((resp) => {
    if (redirectURL) {
      window.location.href = redirectURL;
    } else if (resp.type == "opaqueredirect" && resp.url) {
      window.location.href = resp.url;
    } else {
      window.location.reload();
    }
  });
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
