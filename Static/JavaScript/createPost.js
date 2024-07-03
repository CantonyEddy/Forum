// Static/JavaScript/createPost.js
console.log("createPost.js is loaded");

document.addEventListener('DOMContentLoaded', (event) => {
    document.querySelector('form').addEventListener('submit', function(event) {
        const postName = document.getElementById('postName').value;
        const postMessage = document.getElementById('postMessage').value;
        const category = document.getElementById('category_name').value;

        if (!postName || !postMessage || !category) {
            alert('Please fill out all fields.');
            event.preventDefault();
        }
    });
});

function previewImage(event) {
    const reader = new FileReader();
    reader.onload = function() {
        const imagePreview = document.getElementById('imagePreview');
        imagePreview.src = reader.result;
        imagePreview.style.display = 'block';
    }
    reader.readAsDataURL(event.target.files[0]);
}
