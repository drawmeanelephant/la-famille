document.addEventListener("DOMContentLoaded", () => {
    const toast = document.getElementById("cookie-alert");
    const acceptBtn = document.getElementById("cookie-accept");
    const declineBtn = document.getElementById("cookie-decline");

    if (!toast) return;

    // Check localStorage
    if (localStorage.getItem("cookie_consent") === "dismissed") {
        return; // Already dismissed, stay hidden
    }

    // Show toast
    toast.classList.remove("hidden");

    const dismissToast = () => {
        toast.classList.add("hidden");
        localStorage.setItem("cookie_consent", "dismissed");
    };

    if (acceptBtn) {
        acceptBtn.addEventListener("click", dismissToast);
    }

    if (declineBtn) {
        declineBtn.addEventListener("click", dismissToast);
    }

    // Keyboard support (Escape key)
    document.addEventListener("keydown", (e) => {
        if (e.key === "Escape" && !toast.classList.contains("hidden")) {
            dismissToast();
        }
    });
});
