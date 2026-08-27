document.addEventListener('DOMContentLoaded', function() {
    const inner = document.getElementById('carousel-inner');
    const items = document.querySelectorAll('.carousel-item');
    const prevBtn = document.getElementById('prev-btn');
    const nextBtn = document.getElementById('next-btn');
    
    let index = 0;
    const totalItems = items.length;
    let autoPlayInterval;

    function updateCarousel() {
        inner.style.transition = 'transform 0.5s ease-in-out';
        inner.style.transform = `translateX(${-index * 100}%)`;
    }

    function nextSlide() {
        index = (index + 1) % totalItems;
        updateCarousel();
    }

    function prevSlide() {
        index = (index - 1 + totalItems) % totalItems;
        updateCarousel();
    }

    // Events
    nextBtn.addEventListener('click', () => {
        nextSlide();
        resetAutoPlay();
    });

    prevBtn.addEventListener('click', () => {
        prevSlide();
        resetAutoPlay();
    });

    
    function startAutoPlay() {
        autoPlayInterval = setInterval(nextSlide, 3000);
    }

    function stopAutoPlay() {
        clearInterval(autoPlayInterval);
    }

    function resetAutoPlay() {
        stopAutoPlay();
        startAutoPlay();
    }

    // Pause
    const carousel = document.querySelector('.carousel');
    if (carousel) {
        carousel.addEventListener('mouseenter', stopAutoPlay);
        carousel.addEventListener('mouseleave', startAutoPlay);
    }

    // Starts carousel
    updateCarousel();
    startAutoPlay();
});