/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/FallenProjects/telegram-web-player
 */

package webapp

import (
	"net/http"
)

const webAppHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <title>TgMusic Web Player</title>
    <script src="https://telegram.org/js/telegram-web-app.js"></script>
    <style>
        :root {
            --bg-main: var(--tg-theme-bg-color, #0b0d12);
            --bg-secondary: var(--tg-theme-secondary-bg-color, #14171d);
            --text-main: var(--tg-theme-text-color, #f5f7fa);
            --text-muted: var(--tg-theme-hint-color, #8f98a8);
            --accent-color: var(--tg-theme-button-color, #6d5dfc);
            --accent-text: var(--tg-theme-button-text-color, #fff);
            --primary: #6d5dfc;
            --primary-2: #8b5cf6;
            --primary-glow: rgba(109, 93, 252, .30);
            --glass-bg: rgba(255,255,255,.055);
            --glass-border: rgba(255,255,255,.075);
            --glass-hover: rgba(255,255,255,.09);
            --success: #22c55e;
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
            user-select: none;
            -webkit-user-select: none;
            font-family: -apple-system, BlinkMacSystemFont, "SF Pro Display", "Segoe UI", Roboto, sans-serif;
        }

        html, body {
            min-height: 100%;
            background: #090b0f;
        }

        body {
            color: var(--text-main);
            min-height: 100vh;
            display: flex;
            justify-content: flex-start;
            align-items: center;
            padding: 12px;
            overflow-x: hidden;
            background:
                radial-gradient(700px 420px at 50% -160px, rgba(109,93,252,.22), transparent 68%),
                radial-gradient(500px 350px at 100% 30%, rgba(139,92,246,.08), transparent 70%),
                #090b0f;
        }

        .overlay {
            position: fixed;
            inset: 0;
            background: rgba(7,9,13,.94);
            backdrop-filter: blur(24px);
            -webkit-backdrop-filter: blur(24px);
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            z-index: 1000;
            padding: 24px;
            text-align: center;
            transition: opacity .25s ease, visibility .25s ease;
        }

        .overlay-logo {
            width: 76px;
            height: 76px;
            border-radius: 24px;
            background: linear-gradient(135deg, var(--primary), var(--primary-2));
            display: flex;
            align-items: center;
            justify-content: center;
            margin-bottom: 18px;
            box-shadow: 0 16px 45px var(--primary-glow);
        }

        .overlay-logo svg {
            width: 38px;
            height: 38px;
            fill: #fff;
        }

        .overlay h2 {
            font-size: 23px;
            font-weight: 800;
            letter-spacing: -.5px;
            margin-bottom: 8px;
        }

        .overlay p {
            font-size: 14px;
            color: var(--text-muted);
            margin-bottom: 24px;
            max-width: 320px;
            line-height: 1.5;
        }

        .btn-join {
            background: linear-gradient(135deg, var(--primary), var(--primary-2));
            color: #fff;
            border: 0;
            padding: 14px 26px;
            border-radius: 16px;
            font-size: 15px;
            font-weight: 750;
            box-shadow: 0 12px 30px var(--primary-glow);
            display: flex;
            align-items: center;
            gap: 9px;
            cursor: pointer;
        }

        .btn-join svg {
            width: 20px;
            height: 20px;
            fill: currentColor;
        }

        .btn-join:active { transform: scale(.97); }

        .app-container {
            width: 100%;
            max-width: 430px;
            display: flex;
            flex-direction: column;
            gap: 10px;
            padding-bottom: 12px;
        }

        .top-bar,
        .player-card,
        .queue-card {
            background: var(--glass-bg);
            border: 1px solid var(--glass-border);
            backdrop-filter: blur(22px);
            -webkit-backdrop-filter: blur(22px);
        }

        .top-bar {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 9px 11px;
            border-radius: 18px;
        }

        .user-profile {
            display: flex;
            align-items: center;
            gap: 9px;
            min-width: 0;
        }

        .avatar-container {
            position: relative;
            width: 38px;
            height: 38px;
            flex-shrink: 0;
        }

        .user-avatar {
            width: 100%;
            height: 100%;
            border-radius: 50%;
            object-fit: cover;
            background: linear-gradient(135deg,#3b4250,#20242c);
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 15px;
            font-weight: 750;
            color: #fff;
            border: 1px solid rgba(255,255,255,.12);
        }

        .premium-badge {
            position: absolute;
            right: -2px;
            bottom: -2px;
            width: 15px;
            height: 15px;
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
            background: #11141a;
            color: #facc15;
            font-size: 9px;
            border: 1px solid rgba(255,255,255,.08);
        }

        .premium-badge svg {
            width: 10px;
            height: 10px;
            fill: #facc15;
        }

        .user-info-text {
            min-width: 0;
            display: flex;
            flex-direction: column;
            gap: 1px;
        }

        .user-name-row { min-width: 0; }

        .user-display-name {
            display: block;
            max-width: 150px;
            font-size: 13px;
            font-weight: 700;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .user-handle {
            display: block;
            max-width: 150px;
            font-size: 11px;
            color: var(--text-muted);
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .room-meta {
            display: flex;
            align-items: flex-end;
            flex-direction: column;
            gap: 4px;
            flex-shrink: 0;
        }

        .badge {
            display: flex;
            align-items: center;
            gap: 6px;
            padding: 4px 9px;
            border-radius: 999px;
            background: rgba(255,255,255,.045);
            border: 1px solid var(--glass-border);
            color: var(--text-muted);
            font-size: 10px;
            font-weight: 700;
        }

        .badge.admin {
            color: #86efac;
            background: rgba(34,197,94,.10);
            border-color: rgba(34,197,94,.22);
        }

        .badge-dot {
            width: 6px;
            height: 6px;
            border-radius: 50%;
            background: var(--success);
            box-shadow: 0 0 8px rgba(34,197,94,.6);
        }

        .listeners-trigger {
            display: flex;
            align-items: center;
            gap: 5px;
            padding: 4px 9px;
            border-radius: 999px;
            background: rgba(109,93,252,.12);
            border: 1px solid rgba(109,93,252,.25);
            color: #a79bff;
            font-size: 10px;
            font-weight: 700;
            cursor: pointer;
            transition: background .15s ease, transform .15s ease;
        }

        .listeners-trigger:active {
            transform: scale(.95);
            background: rgba(109,93,252,.22);
        }

        .listeners-trigger svg {
            width: 12px;
            height: 12px;
            fill: currentColor;
        }

        .player-card {
            position: relative;
            overflow: hidden;
            border-radius: 24px;
            padding: 20px 18px 18px;
            display: flex;
            flex-direction: column;
            align-items: center;
            box-shadow: 0 18px 45px rgba(0,0,0,.32);
        }

        .ambient-glow {
            position: absolute;
            top: -90px;
            left: 50%;
            transform: translateX(-50%);
            width: 280px;
            height: 220px;
            background: var(--primary);
            filter: blur(85px);
            opacity: .18;
            border-radius: 50%;
            pointer-events: none;
            transition: opacity .5s ease;
        }

        .artwork-wrapper {
            position: relative;
            width: min(72vw, 260px);
            height: min(72vw, 260px);
            max-width: 260px;
            max-height: 260px;
            margin: 2px 0 18px;
            border-radius: 22px;
            overflow: hidden;
            z-index: 1;
            box-shadow: 0 18px 45px rgba(0,0,0,.48);
        }

        .artwork-img {
            width: 100%;
            height: 100%;
            object-fit: cover;
            border-radius: 22px;
            transition: transform .45s cubic-bezier(.4,0,.2,1);
        }

        .artwork-wrapper.playing .artwork-img { transform: scale(1.025); }

        .platform-badge {
            position: absolute;
            top: 10px;
            right: 10px;
            padding: 5px 8px;
            border-radius: 8px;
            background: rgba(8,10,14,.68);
            backdrop-filter: blur(10px);
            border: 1px solid rgba(255,255,255,.10);
            color: #fff;
            font-size: 9px;
            font-weight: 800;
            letter-spacing: .5px;
        }

        .track-info {
            width: 100%;
            text-align: center;
            margin-bottom: 18px;
            z-index: 1;
        }

        .track-title {
            width: 100%;
            font-size: 20px;
            line-height: 1.2;
            font-weight: 800;
            letter-spacing: -.45px;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
            margin-bottom: 5px;
        }

        .track-artist {
            width: 100%;
            font-size: 13px;
            color: var(--text-muted);
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
            margin-bottom: 10px;
        }

        .requester-badge {
            display: inline-flex;
            align-items: center;
            gap: 6px;
            padding: 5px 10px;
            border-radius: 999px;
            background: rgba(255,255,255,.045);
            border: 1px solid rgba(255,255,255,.07);
            color: var(--text-muted);
            font-size: 10px;
            font-weight: 600;
        }

        .requester-badge svg {
            width: 12px;
            height: 12px;
            fill: currentColor;
        }

        .progress-wrapper {
            width: 100%;
            margin-bottom: 18px;
            z-index: 1;
        }

        .slider-container {
            width: 100%;
            height: 5px;
            display: flex;
            align-items: center;
        }

        .slider {
            width: 100%;
            height: 5px;
            appearance: none;
            -webkit-appearance: none;
            border-radius: 999px;
            background: rgba(255,255,255,.13);
            outline: none;
        }

        .slider::-webkit-slider-thumb {
            appearance: none;
            -webkit-appearance: none;
            width: 14px;
            height: 14px;
            border-radius: 50%;
            background: #fff;
            box-shadow: 0 0 10px var(--primary);
        }

        .time-labels {
            display: flex;
            justify-content: space-between;
            margin-top: 7px;
            font-size: 10px;
            color: var(--text-muted);
            font-variant-numeric: tabular-nums;
        }

        .idle-container {
            width: 100%;
            min-height: 360px;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            text-align: center;
            padding: 28px 12px 20px;
            z-index: 1;
        }

        .idle-icon-box {
            width: 72px;
            height: 72px;
            border-radius: 22px;
            display: flex;
            align-items: center;
            justify-content: center;
            margin-bottom: 17px;
            background: rgba(109,93,252,.10);
            border: 1px solid rgba(109,93,252,.18);
        }

        .idle-svg-icon {
            width: 34px;
            height: 34px;
            fill: #9b91ff;
        }

        .idle-title {
            font-size: 18px;
            font-weight: 800;
            letter-spacing: -.3px;
            margin-bottom: 7px;
        }

        .idle-subtitle {
            max-width: 280px;
            font-size: 12px;
            color: var(--text-muted);
            line-height: 1.5;
            margin-bottom: 16px;
        }

        .cmd-chip {
            padding: 2px 6px;
            border-radius: 6px;
            background: rgba(255,255,255,.06);
            border: 1px solid rgba(255,255,255,.08);
            color: #aaa3ff;
            font-family: monospace;
            font-size: 11px;
        }

        .idle-status-badge {
            display: inline-flex;
            align-items: center;
            gap: 7px;
            padding: 6px 11px;
            border-radius: 999px;
            background: rgba(34,197,94,.08);
            border: 1px solid rgba(34,197,94,.17);
            color: #86efac;
            font-size: 10px;
            font-weight: 700;
        }

        .pulse-dot {
            width: 6px;
            height: 6px;
            border-radius: 50%;
            background: var(--success);
            box-shadow: 0 0 8px var(--success);
            animation: blink 1.8s infinite ease-in-out;
        }

        @keyframes blink {
            0%,100% { opacity: 1; }
            50% { opacity: .4; }
        }

        .controls {
            width: 100%;
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 28px;
            z-index: 1;
        }

        .btn-ctrl {
            width: 48px;
            height: 48px;
            border-radius: 50%;
            border: 1px solid rgba(255,255,255,.09);
            background: rgba(255,255,255,.055);
            color: var(--text-main);
            display: flex;
            align-items: center;
            justify-content: center;
            transition: transform .15s ease, background .15s ease;
            cursor: pointer;
        }

        .btn-ctrl:active:not(:disabled) {
            transform: scale(.91);
            background: rgba(255,255,255,.10);
        }

        .btn-ctrl:disabled { opacity: .28; cursor: not-allowed; }

        .btn-ctrl svg {
            width: 20px;
            height: 20px;
            fill: currentColor;
        }

        .btn-play {
            width: 62px;
            height: 62px;
            border: 0;
            background: linear-gradient(135deg,var(--primary),var(--primary-2));
            box-shadow: 0 10px 28px var(--primary-glow);
        }

        .btn-play svg { width: 25px; height: 25px; }

        .queue-card {
            border-radius: 20px;
            padding: 14px;
        }

        .queue-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin-bottom: 10px;
            padding: 0 2px;
        }

        .queue-title {
            font-size: 11px;
            font-weight: 800;
            letter-spacing: .7px;
            text-transform: uppercase;
            color: var(--text-muted);
        }

        .queue-count {
            font-size: 10px;
            font-weight: 700;
            color: #a9a1ff;
        }

        .queue-list {
            display: flex;
            flex-direction: column;
            gap: 6px;
            max-height: 190px;
            overflow-y: auto;
        }

        .queue-item {
            display: flex;
            align-items: center;
            gap: 10px;
            padding: 8px;
            border-radius: 13px;
            background: rgba(255,255,255,.035);
            border: 1px solid rgba(255,255,255,.045);
        }

        .queue-thumb {
            width: 42px;
            height: 42px;
            border-radius: 9px;
            object-fit: cover;
            flex-shrink: 0;
        }

        .queue-details {
            flex: 1;
            min-width: 0;
        }

        .queue-track-title {
            font-size: 12px;
            font-weight: 650;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .queue-track-sub {
            margin-top: 3px;
            font-size: 10px;
            color: var(--text-muted);
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        /* Drawer for Listeners */
        .modal-backdrop {
            position: fixed;
            inset: 0;
            background: rgba(0, 0, 0, 0.7);
            backdrop-filter: blur(12px);
            -webkit-backdrop-filter: blur(12px);
            z-index: 1500;
            display: none;
            opacity: 0;
            transition: opacity 0.25s ease;
        }

        .modal-drawer {
            position: fixed;
            left: 50%;
            bottom: 0;
            transform: translateX(-50%) translateY(100%);
            width: 100%;
            max-width: 430px;
            max-height: 75vh;
            background: #11141c;
            border-top-left-radius: 24px;
            border-top-right-radius: 24px;
            border: 1px solid var(--glass-border);
            z-index: 1600;
            display: flex;
            flex-direction: column;
            transition: transform 0.3s cubic-bezier(0.16, 1, 0.3, 1);
            box-shadow: 0 -10px 40px rgba(0, 0, 0, 0.6);
        }

        .modal-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 18px 20px 14px;
            border-bottom: 1px solid rgba(255, 255, 255, 0.08);
        }

        .modal-title {
            font-size: 16px;
            font-weight: 800;
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .modal-title svg {
            width: 20px;
            height: 20px;
            fill: var(--primary);
        }

        .modal-close-btn {
            background: rgba(255, 255, 255, 0.08);
            border: 0;
            width: 30px;
            height: 30px;
            border-radius: 50%;
            color: var(--text-muted);
            display: flex;
            align-items: center;
            justify-content: center;
            cursor: pointer;
        }

        .modal-close-btn svg {
            width: 16px;
            height: 16px;
            fill: currentColor;
        }

        .listeners-scroll-list {
            padding: 12px 16px 20px;
            overflow-y: auto;
            display: flex;
            flex-direction: column;
            gap: 10px;
        }

        .listener-item {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 10px 12px;
            border-radius: 14px;
            background: rgba(255, 255, 255, 0.03);
            border: 1px solid rgba(255, 255, 255, 0.05);
        }

        .listener-user-info {
            display: flex;
            align-items: center;
            gap: 12px;
            min-width: 0;
        }

        .listener-avatar {
            width: 36px;
            height: 36px;
            border-radius: 50%;
            object-fit: cover;
            background: linear-gradient(135deg, #3b4250, #20242c);
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: 700;
            font-size: 14px;
            color: #fff;
            flex-shrink: 0;
        }

        .listener-names {
            display: flex;
            flex-direction: column;
            min-width: 0;
        }

        .listener-name {
            font-size: 13px;
            font-weight: 700;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .listener-handle {
            font-size: 11px;
            color: var(--text-muted);
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .toast {
            position: fixed;
            left: 50%;
            bottom: 18px;
            transform: translateX(-50%);
            max-width: calc(100vw - 32px);
            padding: 9px 16px;
            border-radius: 12px;
            background: rgba(20,22,28,.95);
            border: 1px solid rgba(255,255,255,.08);
            color: #fff;
            font-size: 12px;
            font-weight: 650;
            display: none;
            z-index: 2000;
            box-shadow: 0 10px 25px rgba(0,0,0,.35);
            backdrop-filter: blur(12px);
        }

        @media (max-width: 360px) {
            body { padding: 8px; }
            .app-container { gap: 8px; }
            .player-card { padding: 16px 14px; }
            .artwork-wrapper {
                width: min(68vw, 220px);
                height: min(68vw, 220px);
            }
            .controls { gap: 22px; }
        }
    </style>
</head>
<body>

    <div class="overlay" id="no-tg-overlay" style="display: none; z-index: 2000;">
        <div class="overlay-logo" style="background: linear-gradient(135deg, #dc2626, #991b1b);">
            <svg viewBox="0 0 24 24"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/></svg>
        </div>
        <h2>Telegram App Required</h2>
        <p>This Web App player can only be opened inside Telegram. Please open it from your Telegram group or chat.</p>
    </div>

    <div class="overlay" id="join-overlay">
        <div class="overlay-logo">
            <svg viewBox="0 0 24 24"><path d="M12 3v10.55c-.59-.34-1.27-.55-2-.55-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z"/></svg>
        </div>
        <h2>TgMusic Web</h2>
        <p>Join the synchronized audio stream to listen live with group participants.</p>
        <button class="btn-join" id="btn-join">
            <span>Tap to Enter Room</span>
            <svg viewBox="0 0 24 24"><path d="M12 3v10.55c-.59-.34-1.27-.55-2-.55-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z"/></svg>
        </button>
    </div>

    <div class="app-container">
        <!-- Top Navigation Bar -->
        <div class="top-bar">
            <div class="user-profile">
                <div class="avatar-container">
                    <div class="user-avatar" id="user-avatar-placeholder">U</div>
                    <img class="user-avatar" id="user-avatar-img" src="" alt="Avatar" style="display:none;">
                    <div class="premium-badge" id="user-premium-badge" style="display:none;">
                        <svg viewBox="0 0 24 24"><path d="M12 17.27L18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2 9.19 8.63 2 9.24l5.46 4.73L5.82 21z"/></svg>
                    </div>
                </div>
                <div class="user-info-text">
                    <div class="user-name-row">
                        <span class="user-display-name" id="user-display-name">Telegram User</span>
                    </div>
                    <span class="user-handle" id="user-handle">@user</span>
                </div>
            </div>

            <div class="room-meta">
                <div class="badge" id="role-badge">
                    <span class="badge-dot"></span>
                    <span id="role-text">Listener</span>
                </div>
                <div class="listeners-trigger" id="listeners-trigger">
                    <svg viewBox="0 0 24 24"><path d="M16 11c1.66 0 2.99-1.34 2.99-3S17.66 5 16 5s-3 1.34-3 3 1.34 3 3 3zm-8 0c1.66 0 2.99-1.34 2.99-3S9.66 5 8 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V19h14v-2.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.97 1.97 3.45V19h6v-2.5c0-2.33-4.67-3.5-7-3.5z"/></svg>
                    <span id="listeners-count-text">0 Listeners</span>
                </div>
            </div>
        </div>

        <!-- Main Player Card -->
        <div class="player-card">
            <div class="ambient-glow" id="ambient-glow"></div>

            <!-- Idle Empty State View -->
            <div class="idle-container" id="idle-view">
                <div class="idle-icon-box">
                    <svg class="idle-svg-icon" viewBox="0 0 24 24">
                        <path d="M12 3v10.55c-.59-.34-1.27-.55-2-.55-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z"/>
                    </svg>
                </div>
                <div class="idle-title">Nothing Playing Right Now</div>
                <div class="idle-subtitle">
                    Send <span class="cmd-chip">/play song name</span> in Telegram chat to start streaming live music!
                </div>
                <div class="idle-status-badge">
                    <span class="pulse-dot"></span>
                    <span>Ready for Music</span>
                </div>
            </div>

            <!-- Active Player View -->
            <div id="active-player-view" style="display: none; width: 100%; flex-direction: column; align-items: center;">
                <div class="artwork-wrapper" id="art-container">
                    <img id="track-thumb" class="artwork-img" src="" alt="Artwork">
                    <div class="platform-badge" id="platform-badge">MUSIC</div>
                </div>

                <div class="track-info">
                    <div class="track-title" id="track-title">--</div>
                    <div class="track-artist" id="track-artist">--</div>
                    <div class="requester-badge" id="track-requester">
                        <svg viewBox="0 0 24 24"><path d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z"/></svg>
                        <span id="requester-name">Requested by System</span>
                    </div>
                </div>

                <div class="progress-wrapper">
                    <div class="slider-container">
                        <input type="range" class="slider" id="seek-slider" min="0" max="100" value="0" disabled>
                    </div>
                    <div class="time-labels">
                        <span id="curr-time">0:00</span>
                        <span id="total-time">0:00</span>
                    </div>
                </div>

                <div class="controls">
                    <button class="btn-ctrl" id="btn-mute" title="Mute/Unmute">
                        <svg id="icon-vol-high" viewBox="0 0 24 24"><path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z"/></svg>
                        <svg id="icon-vol-mute" viewBox="0 0 24 24" style="display:none;"><path d="M16.5 12c0-1.77-1.02-3.29-2.5-4.03v2.21l2.45 2.45c.03-.2.05-.41.05-.63zm2.5 0c0 .94-.2 1.82-.54 2.64l1.51 1.51C20.63 14.91 21 13.5 21 12c0-4.28-2.99-7.86-7-8.77v2.06c2.89.86 5 3.54 5 6.71zM4.27 3L3 4.27 7.73 9H3v6h4l5 5v-6.73l4.25 4.25c-.67.52-1.42.93-2.25 1.18v2.06c1.38-.31 2.63-.95 3.69-1.81L19.73 21 21 19.73l-9-9L4.27 3zM12 4L9.91 6.09 12 8.18V4z"/></svg>
                    </button>

                    <button class="btn-ctrl btn-play" id="btn-play" disabled>
                        <svg id="icon-play" viewBox="0 0 24 24"><path d="M8 5v14l11-7z"/></svg>
                        <svg id="icon-pause" viewBox="0 0 24 24" style="display:none;"><path d="M6 19h4V5H6v14zm8-14v14h4V5h-4z"/></svg>
                    </button>

                    <button class="btn-ctrl" id="btn-skip" disabled title="Skip Track">
                        <svg viewBox="0 0 24 24"><path d="M6 18l8.5-6L6 6v12zM16 6v12h2V6h-2z"/></svg>
                    </button>
                </div>
            </div>
        </div>

        <!-- Queue Section -->
        <div class="queue-card">
            <div class="queue-header">
                <span class="queue-title">Upcoming Queue</span>
                <span class="queue-count" id="queue-count">0 tracks</span>
            </div>
            <div class="queue-list" id="queue-list">
                <div style="font-size: 12px; color: var(--text-muted); text-align: center; padding: 14px;">No upcoming tracks in queue</div>
            </div>
        </div>
    </div>

    <!-- Drawer for Listeners List -->
    <div class="modal-backdrop" id="modal-backdrop"></div>
    <div class="modal-drawer" id="modal-drawer">
        <div class="modal-header">
            <div class="modal-title">
                <svg viewBox="0 0 24 24"><path d="M16 11c1.66 0 2.99-1.34 2.99-3S17.66 5 16 5s-3 1.34-3 3 1.34 3 3 3zm-8 0c1.66 0 2.99-1.34 2.99-3S9.66 5 8 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V19h14v-2.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.97 1.97 3.45V19h6v-2.5c0-2.33-4.67-3.5-7-3.5z"/></svg>
                <span>Active Listeners (<span id="modal-listeners-count">0</span>)</span>
            </div>
            <button class="modal-close-btn" id="modal-close-btn">
                <svg viewBox="0 0 24 24"><path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/></svg>
            </button>
        </div>
        <div class="listeners-scroll-list" id="listeners-scroll-list">
            <!-- Dynamically populated -->
        </div>
    </div>

    <div class="toast" id="toast-msg"></div>
    <audio id="audio-element" style="display: none;"></audio>

    <script>
        const tg = window.Telegram ? window.Telegram.WebApp : null;
        const noTgOverlay = document.getElementById('no-tg-overlay');

        if (!tg || !tg.initData || tg.initData.trim() === '') {
            if (noTgOverlay) noTgOverlay.style.display = 'flex';
            const joinOv = document.getElementById('join-overlay');
            if (joinOv) joinOv.style.display = 'none';
            throw new Error('Telegram WebApp context required');
        }

        tg.expand();
        tg.ready();

        const urlParams = new URLSearchParams(window.location.search);
        let startParam = (tg.initDataUnsafe && tg.initDataUnsafe.start_param) ? tg.initDataUnsafe.start_param : null;
        if (!startParam) {
            startParam = urlParams.get('tgWebAppStartParam') || urlParams.get('startapp') || urlParams.get('chat_id') || urlParams.get('room');
        }
        let roomId = startParam || '-100000000069';

        const audio = document.getElementById('audio-element');
        const joinOverlay = document.getElementById('join-overlay');
        const btnJoin = document.getElementById('btn-join');
        const idleView = document.getElementById('idle-view');
        const activePlayerView = document.getElementById('active-player-view');
        const trackTitle = document.getElementById('track-title');
        const trackArtist = document.getElementById('track-artist');
        const requesterName = document.getElementById('requester-name');
        const trackThumb = document.getElementById('track-thumb');
        const artContainer = document.getElementById('art-container');
        const platformBadge = document.getElementById('platform-badge');
        const ambientGlow = document.getElementById('ambient-glow');
        const roleBadge = document.getElementById('role-badge');
        const roleText = document.getElementById('role-text');
        const listenersTrigger = document.getElementById('listeners-trigger');
        const listenersCountText = document.getElementById('listeners-count-text');
        const seekSlider = document.getElementById('seek-slider');
        const currTime = document.getElementById('curr-time');
        const totalTime = document.getElementById('total-time');
        const btnPlay = document.getElementById('btn-play');
        const iconPlay = document.getElementById('icon-play');
        const iconPause = document.getElementById('icon-pause');
        const btnSkip = document.getElementById('btn-skip');
        const btnMute = document.getElementById('btn-mute');
        const iconVolHigh = document.getElementById('icon-vol-high');
        const iconVolMute = document.getElementById('icon-vol-mute');
        const queueList = document.getElementById('queue-list');
        const queueCount = document.getElementById('queue-count');
        const toastMsg = document.getElementById('toast-msg');

        // User profile elements
        const userAvatarPlaceholder = document.getElementById('user-avatar-placeholder');
        const userAvatarImg = document.getElementById('user-avatar-img');
        const userPremiumBadge = document.getElementById('user-premium-badge');
        const userDisplayName = document.getElementById('user-display-name');
        const userHandle = document.getElementById('user-handle');

        // Modal elements
        const modalBackdrop = document.getElementById('modal-backdrop');
        const modalDrawer = document.getElementById('modal-drawer');
        const modalCloseBtn = document.getElementById('modal-close-btn');
        const modalListenersCount = document.getElementById('modal-listeners-count');
        const listenersScrollList = document.getElementById('listeners-scroll-list');

        let trackDuration = 0;
        let currentPosition = 0;
        let serverTimeOffset = 0;
        let roomState = null;
        let isUserSeeking = false;
        let isAdmin = false;
        let canControl = false;
        let ws = null;
        let isAudioUnlocked = false;
        let pendingSeekPosition = null;
        let playPromise = null;

        function startAudioPlayback() {
            if (!audio.src) return;
            const promise = audio.play();
            if (promise !== undefined) {
                playPromise = promise;
                promise.then(() => {
                    if (playPromise === promise) playPromise = null;
                    if (!roomState || !roomState.track) {
                        stopAudioPlayback(true);
                    } else if (roomState.playback && roomState.playback.status !== 'playing') {
                        stopAudioPlayback(false);
                    }
                }).catch(e => {
                    if (playPromise === promise) playPromise = null;
                    console.log('Playback error:', e);
                });
            }
        }

        function stopAudioPlayback(fullStop) {
            audio.pause();
            if (fullStop) {
                audio.src = '';
                audio.removeAttribute('src');
                audio.load();
            }
        }

        audio.addEventListener('loadedmetadata', () => {
            if (pendingSeekPosition !== null && audio.readyState >= 1) {
                audio.currentTime = pendingSeekPosition;
                pendingSeekPosition = null;
            }
        });

        function triggerHaptic(style) {
            if (tg && tg.HapticFeedback) {
                tg.HapticFeedback.impactOccurred(style || 'light');
            }
        }

        function populateUserProfile() {
            if (!tg || !tg.initDataUnsafe) return;
            const u = tg.initDataUnsafe.user;
            if (u) {
                const name = (u.first_name || '') + (u.last_name ? ' ' + u.last_name : '');
                userDisplayName.innerText = name || 'Telegram User';
                userHandle.innerText = u.username ? '@' + u.username : 'ID: ' + u.id;

                if (u.photo_url) {
                    userAvatarImg.src = u.photo_url;
                    userAvatarImg.style.display = 'block';
                    userAvatarPlaceholder.style.display = 'none';
                } else {
                    const initials = (u.first_name ? u.first_name[0] : 'U').toUpperCase();
                    userAvatarPlaceholder.innerText = initials;
                }

                if (u.is_premium) {
                    userPremiumBadge.style.display = 'flex';
                }
            }
        }

        populateUserProfile();

        function showToast(msg) {
            toastMsg.innerText = msg;
            toastMsg.style.display = 'block';
            setTimeout(() => { toastMsg.style.display = 'none'; }, 3000);
        }

        function formatTime(secs) {
            secs = Math.floor(secs || 0);
            const m = Math.floor(secs / 60);
            const s = secs % 60;
            return m + ':' + (s < 10 ? '0' : '') + s;
        }

        function openListenersModal() {
            triggerHaptic('light');
            modalBackdrop.style.display = 'block';
            setTimeout(() => {
                modalBackdrop.style.opacity = '1';
                modalDrawer.style.transform = 'translateX(-50%) translateY(0)';
            }, 10);
        }

        function closeListenersModal() {
            triggerHaptic('light');
            modalBackdrop.style.opacity = '0';
            modalDrawer.style.transform = 'translateX(-50%) translateY(100%)';
            setTimeout(() => {
                modalBackdrop.style.display = 'none';
            }, 300);
        }

        listenersTrigger.addEventListener('click', openListenersModal);
        modalCloseBtn.addEventListener('click', closeListenersModal);
        modalBackdrop.addEventListener('click', closeListenersModal);

        btnJoin.addEventListener('click', () => {
            triggerHaptic('medium');
            isAudioUnlocked = true;
            audio.src = 'data:audio/wav;base64,UklGRiQAAABXQVZFZm10IBAAAAABAAEARKwAAIhYAQACABAAZGF0YQAAAAA=';
            startAudioPlayback();
            joinOverlay.style.opacity = '0';
            joinOverlay.style.visibility = 'hidden';
            setTimeout(() => { joinOverlay.style.display = 'none'; }, 300);
            if (roomState) {
                updateRoomState(roomState);
            }
        });

        connectWS();

        audio.addEventListener('ended', () => {
            if (ws && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({ type: 'track_end' }));
            }
        });

        function connectWS() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = protocol + '//' + window.location.host + '/ws?room=' + roomId;
            ws = new WebSocket(wsUrl);

            ws.onopen = () => {
                ws.send(JSON.stringify({
                    type: 'join',
                    roomId: roomId,
                    initData: tg ? tg.initData : ''
                }));
                pingServer();
            };

            ws.onmessage = (event) => {
                try {
                    const msg = JSON.parse(event.data);
                    if (msg.event === 'room_state') {
                        updateRoomState(msg.data);
                    } else if (msg.event === 'user_info') {
                        isAdmin = msg.data.isAdmin;
                        canControl = msg.data.canControl !== undefined ? msg.data.canControl : isAdmin;
                        if (isAdmin) {
                            roleBadge.classList.add('admin');
                            roleText.innerText = 'Admin';
                        } else {
                            roleBadge.classList.remove('admin');
                            roleText.innerText = 'Listener';
                        }
                        updateControlButtonsState();
                    } else if (msg.event === 'pong') {
                        const now = Date.now();
                        const rtt = now - msg.data.clientTime;
                        const serverTime = msg.data.serverTime + (rtt / 2);
                        serverTimeOffset = serverTime - now;
                    } else if (msg.event === 'error') {
                        showToast(msg.data);
                    }
                } catch (e) {
                    console.error(e);
                }
            };

            ws.onclose = () => {
                setTimeout(connectWS, 2000);
            };
        }

        function pingServer() {
            if (ws && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({ type: 'ping', clientTime: Date.now() }));
            }
        }

        setInterval(pingServer, 10000);

        function updateControlButtonsState() {
            btnPlay.disabled = !canControl;
            btnSkip.disabled = !canControl;
            seekSlider.disabled = !canControl;
        }

        function updateRoomState(data) {
            roomState = data;
            const listenersCount = data.listeners ? data.listeners.length : 0;
            listenersCountText.innerText = listenersCount + (listenersCount === 1 ? ' Listener' : ' Listeners');
            modalListenersCount.innerText = listenersCount;

            updateListenersList(data.listeners || []);

            const track = data.track;
            const pb = data.playback;

            if (!track) {
                idleView.style.display = 'flex';
                activePlayerView.style.display = 'none';
                ambientGlow.style.opacity = '0.1';
                iconPlay.style.display = 'block';
                iconPause.style.display = 'none';
                stopAudioPlayback(true);
                currTime.innerText = '0:00';
                totalTime.innerText = '0:00';
                seekSlider.value = 0;
                updateQueue([]);
                return;
            }

            idleView.style.display = 'none';
            activePlayerView.style.display = 'flex';

            trackTitle.innerText = track.title || 'Unknown Track';
            trackArtist.innerText = track.artist || track.platform || 'Music';
            requesterName.innerText = 'Requested by ' + (track.user || 'User');
            platformBadge.innerText = (track.platform || 'Music').toUpperCase();
            if (track.thumbnail) {
                trackThumb.src = track.thumbnail;
            } else {
                trackThumb.src = 'https://i.pinimg.com/736x/0d/f4/65/0df465d1e98239ecb6283400605fc813.jpg';
            }
            trackDuration = track.duration || 0;
            totalTime.innerText = formatTime(trackDuration);

            const audioSrc = track.audioUrl;
            let srcChanged = false;
            if (audioSrc && audio.src !== window.location.origin + audioSrc && !audio.src.endsWith(audioSrc)) {
                audio.src = audioSrc;
                srcChanged = true;
            }

            let targetPos = pb.position || 0;
            if (pb.status === 'playing') {
                const nowServer = Date.now() + serverTimeOffset;
                const elapsed = (nowServer - pb.serverTime) / 1000;
                targetPos += elapsed;
            }

            if (targetPos > trackDuration) targetPos = trackDuration;

            currentPosition = targetPos;
            if (!isUserSeeking) {
                seekSlider.max = trackDuration;
                seekSlider.value = currentPosition;
                currTime.innerText = formatTime(currentPosition);
            }

            if (pb.status === 'playing') {
                iconPlay.style.display = 'none';
                iconPause.style.display = 'block';
                artContainer.classList.add('playing');
                ambientGlow.style.opacity = '0.35';
                if (isAudioUnlocked) {
                    if (srcChanged) {
                        pendingSeekPosition = targetPos;
                        startAudioPlayback();
                    } else {
                        if (Math.abs(audio.currentTime - targetPos) > 0.8 && audio.readyState >= 1) {
                            audio.currentTime = targetPos;
                        }
                        if (audio.paused) {
                            startAudioPlayback();
                        }
                    }
                }
            } else {
                iconPlay.style.display = 'block';
                iconPause.style.display = 'none';
                artContainer.classList.remove('playing');
                ambientGlow.style.opacity = '0.1';
                stopAudioPlayback(false);
                if (audio.readyState >= 1) {
                    audio.currentTime = targetPos;
                }
            }

            updateQueue(data.queue || []);
        }

        function updateListenersList(listeners) {
            if (!listeners || listeners.length === 0) {
                listenersScrollList.innerHTML = '<div style="font-size: 13px; color: var(--text-muted); text-align: center; padding: 20px;">No active listeners connected.</div>';
                return;
            }

            let html = '';
            listeners.forEach(l => {
                const fullName = (l.firstName || 'Listener') + (l.lastName ? ' ' + l.lastName : '');
                const handle = l.username ? '@' + l.username : (l.userId ? 'ID: ' + l.userId : 'Anonymous');
                const initial = (l.firstName ? l.firstName[0] : 'L').toUpperCase();

                html += '<div class="listener-item">';
                html += '<div class="listener-user-info">';
                if (l.photoUrl) {
                    html += '<img class="listener-avatar" src="' + l.photoUrl + '" alt="avatar">';
                } else {
                    html += '<div class="listener-avatar">' + initial + '</div>';
                }
                html += '<div class="listener-names">';
                html += '<span class="listener-name">' + fullName + '</span>';
                html += '<span class="listener-handle">' + handle + '</span>';
                html += '</div></div>';

                if (l.isAdmin) {
                    html += '<div class="badge admin"><span class="badge-dot"></span><span>Admin</span></div>';
                } else {
                    html += '<div class="badge"><span id="role-text">Listener</span></div>';
                }
                html += '</div>';
            });
            listenersScrollList.innerHTML = html;
        }

        function updateQueue(queue) {
            queueCount.innerText = queue.length + ' tracks';
            if (!queue || queue.length === 0) {
                queueList.innerHTML = '<div style="font-size: 12px; color: var(--text-muted); text-align: center; padding: 14px;">No upcoming tracks in queue</div>';
                return;
            }

            let html = '';
            queue.forEach((item, index) => {
                html += '<div class="queue-item">';
                html += '<img class="queue-thumb" src="' + (item.thumbnail || trackThumb.src) + '" alt="thumb">';
                html += '<div class="queue-details">';
                html += '<div class="queue-track-title">' + (index + 1) + '. ' + (item.title || 'Track') + '</div>';
                html += '<div class="queue-track-sub">' + formatTime(item.duration) + ' • Requested by ' + (item.user || 'User') + '</div>';
                html += '</div></div>';
            });
            queueList.innerHTML = html;
        }

        btnPlay.addEventListener('click', () => {
            triggerHaptic('light');
            if (!canControl || !roomState || !roomState.track) return;
            if (roomState.playback.status === 'playing') {
                ws.send(JSON.stringify({ type: 'pause' }));
            } else {
                ws.send(JSON.stringify({ type: 'resume' }));
            }
        });

        btnSkip.addEventListener('click', () => {
            triggerHaptic('medium');
            if (!canControl) return;
            ws.send(JSON.stringify({ type: 'skip' }));
        });

        btnMute.addEventListener('click', () => {
            triggerHaptic('light');
            audio.muted = !audio.muted;
            if (audio.muted) {
                iconVolHigh.style.display = 'none';
                iconVolMute.style.display = 'block';
            } else {
                iconVolHigh.style.display = 'block';
                iconVolMute.style.display = 'none';
            }
        });

        seekSlider.addEventListener('input', () => {
            if (!canControl) return;
            isUserSeeking = true;
            currTime.innerText = formatTime(seekSlider.value);
        });

        seekSlider.addEventListener('change', () => {
            if (!canControl) return;
            triggerHaptic('light');
            isUserSeeking = false;
            const pos = parseFloat(seekSlider.value);
            ws.send(JSON.stringify({ type: 'seek', positionSeconds: pos }));
        });

        setInterval(() => {
            if (roomState && roomState.playback && roomState.playback.status === 'playing' && !isUserSeeking) {
                currentPosition += 0.5;
                if (currentPosition > trackDuration) currentPosition = trackDuration;
                seekSlider.value = currentPosition;
                currTime.innerText = formatTime(currentPosition);
            }
        }, 500);
    </script>
</body>
</html>`

func serveWebAppHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(webAppHTML))
}
