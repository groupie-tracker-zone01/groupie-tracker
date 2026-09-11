document.addEventListener('DOMContentLoaded', function() {
    const widgets = document.querySelectorAll('[data-search-widget]');
    const maxResults = 8;

    function normalize(value) {
        return value
            .toLocaleLowerCase()
            .normalize('NFD')
            .replace(/[\u0300-\u036f]/g, '')
            .replace(/[_,-]+/g, ' ')
            .replace(/\s+/g, ' ')
            .trim();
    }

    widgets.forEach(function(widget) {
        const input = widget.querySelector('input[type="search"]');
        const dropdown = widget.querySelector('.dropdown-results');
        const options = Array.from(widget.querySelectorAll('.dropdown-option'));

        if (!input || !dropdown || options.length === 0) {
            return;
        }

        function visibleOptions() {
            return options.filter(function(option) {
                return !option.closest('li').hidden;
            });
        }

        function closeDropdown() {
            dropdown.hidden = true;
            input.setAttribute('aria-expanded', 'false');
            options.forEach(function(option) {
                option.classList.remove('is-active');
            });
        }

        function openDropdown() {
            dropdown.hidden = false;
            input.setAttribute('aria-expanded', 'true');
        }

        function filterOptions() {
            const query = normalize(input.value);

            if (query === '') {
                closeDropdown();
                return;
            }

            const seen = new Set();
            let shown = 0;

            options.forEach(function(option) {
                const item = option.closest('li');
                const value = normalize(option.dataset.value || option.textContent);
                const subtitle = normalize(
                    option.querySelector('.dropdown-subtitle')?.textContent || ''
                );
                const key = value + '|' + subtitle;
                const matches = value.includes(query);

                if (matches && !seen.has(key) && shown < maxResults) {
                    item.hidden = false;
                    seen.add(key);
                    shown++;
                } else {
                    item.hidden = true;
                }

                option.classList.remove('is-active');
            });

            if (shown > 0) {
                openDropdown();
            } else {
                closeDropdown();
            }
        }

        function selectOption(option) {
            input.value = option.dataset.value || '';
            closeDropdown();
            input.focus();
        }

        input.addEventListener('input', filterOptions);

        input.addEventListener('focus', function() {
            if (input.value.trim() !== '') {
                filterOptions();
            }
        });

        input.addEventListener('keydown', function(event) {
            if (event.key === 'Escape') {
                closeDropdown();
                return;
            }

            if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp') {
                return;
            }

            const visible = visibleOptions();
            if (visible.length === 0) {
                return;
            }

            event.preventDefault();
            const target = event.key === 'ArrowDown'
                ? visible[0]
                : visible[visible.length - 1];

            target.classList.add('is-active');
            target.focus();
        });

        options.forEach(function(option) {
            option.addEventListener('click', function() {
                selectOption(option);
            });

            option.addEventListener('keydown', function(event) {
                if (event.key === 'Escape') {
                    event.preventDefault();
                    closeDropdown();
                    input.focus();
                    return;
                }

                if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp') {
                    return;
                }

                const visible = visibleOptions();
                const index = visible.indexOf(option);

                if (index === -1) {
                    return;
                }

                event.preventDefault();
                option.classList.remove('is-active');

                const nextIndex = event.key === 'ArrowDown'
                    ? (index + 1) % visible.length
                    : (index - 1 + visible.length) % visible.length;

                visible[nextIndex].classList.add('is-active');
                visible[nextIndex].focus();
            });
        });

        document.addEventListener('pointerdown', function(event) {
            if (!widget.contains(event.target)) {
                closeDropdown();
            }
        });
    });
});
