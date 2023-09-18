// import 'sanitize.css';
import '@fortawesome/fontawesome-free/js/brands.js'
import '@fortawesome/fontawesome-free/js/solid.js'
import '@fortawesome/fontawesome-free/js/fontawesome.js'
import './css/app.scss';
import Alpine from 'alpinejs';

window["Alpine"] = Alpine;

window.addEventListener('DOMContentLoaded', () => {
    Alpine.start();
});
