// createPost.js
document.querySelector('form').addEventListener('submit', function(event) {
    const postName = document.getElementById('postName').value;
    const postMessage = document.getElementById('postMessage').value;
    const category = document.getElementById('category_name').value;

    if (!postName || !postMessage || !category) {
        alert('Please fill out all fields.');
        event.preventDefault();
    }
});