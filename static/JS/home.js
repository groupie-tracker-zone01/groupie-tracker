document.addEventListener('DOMContentLoaded', function() {
    const carousel = document.getElementById('carousel');
    const inner = document.getElementById('carousel-inner');
    const items = document.querySelectorAll('.carousel-item');
    const prevBtn = document.getElementById('prev-btn');
    const nextBtn = document.getElementById('next-btn');
    const toggleBtn = document.getElementById('carousel-toggle');

    if (!carousel || !inner || !prevBtn || !nextBtn || !toggleBtn || items.length === 0) {
        return;
    }

    let index = 0;
    let autoPlayInterval;
    let pausedByUser = false;
    const totalItems = items.length;
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

    function updateCarousel() {
        inner.style.transform = `translateX(${-index * 100}%)`;
        items.forEach((item, itemIndex) => {
            item.classList.toggle('active', itemIndex === index);
            item.setAttribute('aria-hidden', itemIndex === index ? 'false' : 'true');
        });
    }

    function nextSlide() {
        index = (index + 1) % totalItems;
        updateCarousel();
    }

    function prevSlide() {
        index = (index - 1 + totalItems) % totalItems;
        updateCarousel();
    }

    function stopAutoPlay() {
        window.clearInterval(autoPlayInterval);
        autoPlayInterval = undefined;
    }

    function startAutoPlay() {
        stopAutoPlay();
        if (!pausedByUser && !reducedMotion && !document.hidden) {
            autoPlayInterval = window.setInterval(nextSlide, 5000);
        }
    }

    function setPausedByUser(paused) {
        pausedByUser = paused;
        toggleBtn.textContent = paused ? 'Play' : 'Pause';
        toggleBtn.setAttribute('aria-label', paused ? 'Play carousel' : 'Pause carousel');
        toggleBtn.setAttribute('aria-pressed', paused ? 'true' : 'false');
        if (paused) {
            stopAutoPlay();
        } else {
            startAutoPlay();
        }
    }

    nextBtn.addEventListener('click', function() {
        nextSlide();
        startAutoPlay();
    });

    prevBtn.addEventListener('click', function() {
        prevSlide();
        startAutoPlay();
    });

    toggleBtn.addEventListener('click', function() {
        setPausedByUser(!pausedByUser);
    });

    carousel.addEventListener('keydown', function(event) {
        if (event.key === 'ArrowRight') {
            event.preventDefault();
            nextSlide();
            startAutoPlay();
        } else if (event.key === 'ArrowLeft') {
            event.preventDefault();
            prevSlide();
            startAutoPlay();
        }
    });

    // Stop movement while the user is reading or interacting with the carousel.
    carousel.addEventListener('mouseenter', stopAutoPlay);
    carousel.addEventListener('mouseleave', startAutoPlay);
    carousel.addEventListener('focusin', stopAutoPlay);
    carousel.addEventListener('focusout', startAutoPlay);

    document.addEventListener('visibilitychange', function() {
        if (document.hidden) {
            stopAutoPlay();
        } else {
            startAutoPlay();
        }
    });

    updateCarousel();
    if (reducedMotion) {
        setPausedByUser(true);
    } else {
        startAutoPlay();
    }
});
