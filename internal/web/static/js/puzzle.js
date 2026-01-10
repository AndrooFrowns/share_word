window.calculatePanDeltas = (el, sidebarOpen) => {
    const stage = document.getElementById('puzzle-stage');
    if (!stage || !el) return {dx: 0, dy: 0};
    const stageRect = stage.getBoundingClientRect();
    const cellRect = el.getBoundingClientRect();
    
    // Focus bar is approx 80-100px.
    const barHeight = 100;
    // Sidebar is approx 350px.
    const sidebarWidth = sidebarOpen ? 350 : 0;
    
    // We want to center in the visible area.
    const targetX = stageRect.left + (stageRect.width - sidebarWidth) / 2;
    const targetY = stageRect.top + (stageRect.height - barHeight) / 2;
    
    return {
        dx: targetX - (cellRect.left + cellRect.width / 2),
        dy: targetY - (cellRect.top + cellRect.height / 2)
    };
};
