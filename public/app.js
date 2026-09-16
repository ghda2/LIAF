// The LIAF Chronicle — Interatividade Monocromática e Controle de Tema
document.addEventListener("DOMContentLoaded", () => {
  const themeBtn = document.getElementById("theme-btn");
  const html = document.documentElement;

  // Carrega tema salvo ou preferência do sistema
  const savedTheme = localStorage.getItem("liaf_chronicle_theme");
  if (savedTheme) {
    html.setAttribute("data-theme", savedTheme);
    updateThemeButtonText(savedTheme);
  } else if (window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches) {
    html.setAttribute("data-theme", "dark");
    updateThemeButtonText("dark");
  }

  if (themeBtn) {
    themeBtn.addEventListener("click", () => {
      const current = html.getAttribute("data-theme") || "light";
      const next = current === "light" ? "dark" : "light";
      html.setAttribute("data-theme", next);
      localStorage.setItem("liaf_chronicle_theme", next);
      updateThemeButtonText(next);
    });
  }

  function updateThemeButtonText(theme) {
    if (themeBtn) {
      themeBtn.textContent = theme === "dark" ? "Modo Claro" : "Modo Escuro";
    }
  }

  // Atualiza data editorial se o elemento existir
  const dateEl = document.getElementById("edition-date");
  if (dateEl) {
    const now = new Date();
    const options = { year: "numeric", month: "long", day: "numeric" };
    dateEl.textContent = now.toLocaleDateString("pt-BR", options).toUpperCase();
  }
});
