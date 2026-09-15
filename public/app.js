// Script de interatividade do LIAF Web Engine
// Comentários e quebras de linha serão minificados!

document.addEventListener("DOMContentLoaded", function() {
  /* Listener do botão */
  const btn = document.getElementById("btn");
  const output = document.getElementById("output");

  if (btn && output) {
    btn.addEventListener("click", function() {
      output.innerText = "✓ JavaScript carregado e minificado com sucesso pelo LIAF!";
      output.style.color = "#4ade80";
    });
  }
});
