function showNotification({ text, color }: any) {
  const notification = document.getElementById("notification");
  if (!notification) {
    return;
  }
  notification.innerText = text;
  notification.style.backgroundColor = color;
  notification.style.display = "block";
  notification.classList.add("scale-0");
  setTimeout(() => {
    notification.classList.remove("scale-0");
    notification.style.display = "none";
  }, 1500);
}
export { showNotification };
