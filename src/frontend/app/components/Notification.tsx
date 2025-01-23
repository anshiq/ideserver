const Notify = () => {
  const notify = document.getElementById("notification");
  if (notify) {
    alert("hi");
    notify.style.display = "flex";
    notify.innerText = "Anshik Singh";
  }
};
export { Notify };
