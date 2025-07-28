// FLV播放器初始化函数
function initFlvPlayer(videoElement, url) {
    if (flvjs.isSupported()) {
        var flvPlayer = flvjs.createPlayer({
            type: 'flv',
            url: url
        });
        flvPlayer.attachMediaElement(videoElement);
        flvPlayer.load();
        
        // 返回播放器实例，以便后续可以销毁
        return flvPlayer;
    } else {
        console.error('您的浏览器不支持FLV播放');
        videoElement.parentNode.innerHTML = '<div style="color: red; text-align: center; padding: 20px;">您的浏览器不支持FLV播放，请使用Chrome、Firefox或Edge浏览器</div>';
        return null;
    }
}

// 销毁播放器实例
function destroyFlvPlayer(player) {
    if (player) {
        player.pause();
        player.unload();
        player.detachMediaElement();
        player.destroy();
        return true;
    }
    return false;
}