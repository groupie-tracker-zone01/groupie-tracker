document.addEventListener('DOMContentLoaded', function() {
    const inner = document.getElementById('carousel-inner');
    const items = document.querySelectorAll('.carousel-item');
    const prevBtn = document.getElementById('prev-btn');
    const nextBtn = document.getElementById('next-btn');
    
    let index = 0;
    const totalItems = items.length;
    let autoPlayInterval; // Variável para guardar o intervalo

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

    // Adiciona eventos
    nextBtn.addEventListener('click', () => {
        nextSlide();
        resetAutoPlay(); // Reinicia o timer quando o usuário clica
    });

    prevBtn.addEventListener('click', () => {
        prevSlide();
        resetAutoPlay(); // Reinicia o timer quando o usuário clica
    });

    // ==========================================
    // LÓGICA DO AUTO-PLAY (Troca automática)
    // ==========================================
    function startAutoPlay() {
        autoPlayInterval = setInterval(nextSlide, 3000); // Troca a cada 3 segundos (3000ms)
    }

    function stopAutoPlay() {
        clearInterval(autoPlayInterval);
    }

    function resetAutoPlay() {
        stopAutoPlay();
        startAutoPlay();
    }

    // Pausa quando o mouse está em cima do carrossel
    const carousel = document.querySelector('.carousel');
    if (carousel) {
        carousel.addEventListener('mouseenter', stopAutoPlay);
        carousel.addEventListener('mouseleave', startAutoPlay);
    }

    // Inicializa o carrossel
    updateCarousel();
    startAutoPlay(); // Começa a troca automática assim que a página carrega
});