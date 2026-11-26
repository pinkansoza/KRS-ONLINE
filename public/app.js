// Konfirmasi sebelum aksi berbahaya. Ditulis di file terpisah karena
// Content-Security-Policy melarang script inline.
document.addEventListener("submit", function (e) {
  var pesan = e.target.getAttribute("data-confirm");
  if (pesan && !window.confirm(pesan)) {
    e.preventDefault();
  }
});

// Tombol cetak -> simpan sebagai PDF dari dialog print browser.
document.addEventListener("click", function (e) {
  if (e.target.matches("[data-print]")) {
    window.print();
  }
});