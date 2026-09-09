// Colors pour apply
    const colors = [
        // { bg: '#cd0101', text: '#fff' },
        // { bg: '#f8fbfb', text: '#0a0a0a' },
        { bg: '#1f2323', text: '#fff' },
        { bg: 'hsl(46, 100%, 38%)', text: '#181919' },
    ];

    // Looping
    function applyColorsToDates(selector) {
        document.querySelectorAll(selector).forEach((el, index) => {
            const colorIndex = index % colors.length;
            el.style.backgroundColor = colors[colorIndex].bg;
            el.style.color = colors[colorIndex].text;
        });
    }
    applyColorsToDates('.date-tag');
    applyColorsToDates('.relations-dates-tag');
    
    const colorsText = [
        // { bg: '#cd0101', text: '#fff' },
        // { bg: '#f8fbfb', text: '#0a0a0a' },
    
        { text: 'hsl(46, 100%, 38%)' },
        { text: '#918d8d' },
    ];
    
    function applyColorsToCity(selector) {
        document.querySelectorAll(selector).forEach((el, index) => {
            const colorIndex = index % colorsText.length;
            el.style.backgroundColor = colorsText[colorIndex].bg;
            el.style.color = colorsText[colorIndex].text;
        });
    }
    applyColorsToCity('.relations-city');