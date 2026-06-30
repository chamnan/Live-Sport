// ===== IPTV Sports Player =====

// ===== STATE =====
let art = null;
let hls = null;
let activeMatchId = null;
let isMobile = window.innerWidth < 768;

// Suppress all unhandled promise rejections from Artplayer's internal play() calls
window.addEventListener('unhandledrejection', (e) => {
    e.preventDefault();
});

// ===== UTILS =====
function formatDisplayDate(yyyymmdd) {
    const y = yyyymmdd.slice(0, 4);
    const m = yyyymmdd.slice(4, 6);
    const d = yyyymmdd.slice(6, 8);
    const date = new Date(y, m - 1, d);
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    const diff = Math.round((date - today) / 86400000);
    const label = diff === -1 ? '📅 Yesterday' : diff === 0 ? '📅 Today' : diff === 1 ? '📅 Tomorrow' : yyyymmdd;
    return label + ' — ' + d + '/' + m + '/' + y;
}

// ===== TOAST =====
function showToast(msg, type) {
    type = type || 'info';
    const icons = { error: '⚠️', warn: '🔔', info: 'ℹ️' };
    const container = document.getElementById('toast-container');

    const toast = document.createElement('div');
    toast.className = 'toast ' + type;
    toast.innerHTML = '<span class="toast-icon">' + (icons[type] || '') + '</span>' + msg;

    // Remove after animation ends (3s total)
    toast.addEventListener('animationend', function(e) {
        if (e.animationName === 'toastOut') toast.remove();
    });

    container.appendChild(toast);

    // Limit to 3 toasts at a time
    while (container.children.length > 3) {
        container.firstChild.remove();
    }
}

// ===== MOBILE HELPERS =====
function isMobileDevice() {
    return window.innerWidth < 768;
}

function toggleSidebar() {
    const sidebar = document.getElementById('sidebar');
    const overlay = document.getElementById('sidebar-overlay');
    const open = !sidebar.classList.contains('open');
    sidebar.classList.toggle('open', open);
    overlay.classList.toggle('show', open);
}

function closeSidebar() {
    document.getElementById('sidebar').classList.remove('open');
    document.getElementById('sidebar-overlay').classList.remove('show');
}

// ===== TIME FORMATTING =====
function formatMatchTime(raw) {
    if (raw == null || raw === '') return '';
    let h, m;
    // Unix timestamp (seconds or milliseconds)
    if (typeof raw === 'number') {
        const d = raw > 1e12 ? new Date(raw) : new Date(raw * 1000);
        h = d.getHours();
        m = d.getMinutes();
    } else {
        const str = String(raw);
        // ISO date string like "2026-06-29T14:30:00"
        const isoMatch = str.match(/T(\d{2}):(\d{2})/);
        if (isoMatch) {
            h = parseInt(isoMatch[1], 10);
            m = parseInt(isoMatch[2], 10);
        } else {
            // Plain time like "14:30" or "14:30:00"
            const parts = str.split(':');
            h = parseInt(parts[0], 10);
            m = parseInt(parts[1], 10);
            if (isNaN(h) || isNaN(m)) return '';
        }
    }
    const ampm = h >= 12 ? 'PM' : 'AM';
    const h12 = h % 12 || 12;
    const mm = String(m).padStart(2, '0');
    return h12 + ':' + mm + ' ' + ampm;
}

// ===== PLAYER =====
function destroyPlayer() {
    if (hls) { hls.destroy(); hls = null; }
    if (art) { art.destroy(false); art = null; }
    hideOverlay();
}

function showOverlay(homeName, awayName, homeLogo, awayLogo, scoreStr) {
    const overlay = document.getElementById('match-overlay');
    overlay.style.display = 'flex';

    const homeImg = document.getElementById('overlay-home-logo');
    homeImg.src = homeLogo || '';
    homeImg.style.display = homeLogo ? 'block' : 'none';

    const awayImg = document.getElementById('overlay-away-logo');
    awayImg.src = awayLogo || '';
    awayImg.style.display = awayLogo ? 'block' : 'none';

    document.getElementById('overlay-teams').innerText = homeName + '  vs  ' + awayName;
    document.getElementById('overlay-score').innerText = scoreStr || '';
}

function hideOverlay() {
    document.getElementById('match-overlay').style.display = 'none';
}

/**
 * PLAY STREAM
 */
async function play(streamUrl, matchId, home, away, homeLogo, awayLogo, scoreStr) {
    const pr = await fetch(
        `/api/play-url?u=${encodeURIComponent(streamUrl)}&matchId=${matchId}&home=${encodeURIComponent(home)}&away=${encodeURIComponent(away)}`
    );
    const pj = await pr.json();

    if (!pj.url) return;

    destroyPlayer();

    art = new Artplayer({
        container: '#player',
        url: pj.url,
        isLive: true,
        autoplay: true,
        muted: true,
        autoMini: true,
        fullscreen: true,
        flip: true,
        playbackRate: true,
    });

    showOverlay(home, away, homeLogo, awayLogo, scoreStr);
}

// ===== DATA LOADING =====
async function loadMatches() {
    const res = await fetch('/api/matches');
    const json = await res.json();
    const result = json.result || {};
    const days = json.days || Object.keys(result).sort();

    // Build a map: day -> normalized groups with dedup
    const dayGroups = {};

    days.forEach(day => {
        let data = result[day];
        if (!data || data.error) return;

        // Unwrap extra .data wrapper if present
        if (data.data && typeof data.data === 'object' && !Array.isArray(data.data)) {
            data = data.data;
        }

        const groups = normalizeGroups(data);
        // Ensure living_group (🔴) always takes priority during dedup
        groups.sort((a, b) => {
            const aLive = a.title.startsWith('🔴') ? 1 : 0;
            const bLive = b.title.startsWith('🔴') ? 1 : 0;
            return bLive - aLive; // living groups sort first
        });
        // Dedup: track seen competition names + match IDs per day
        const seenComp = new Set();
        const seenMatch = new Set();
        const deduped = [];

        groups.forEach(g => {
            const compKey = g.competitionName || g.title;
            if (seenComp.has(compKey)) return; // skip duplicate competition
            seenComp.add(compKey);

            // Filter items by deduplicating match IDs inside match arrays
            const uniqueItems = g.items.filter(item => {
                const matchArr = item.match;
                if (!Array.isArray(matchArr) || matchArr.length === 0) return true;

                // Keep only matches whose IDs haven't been seen
                const uniqueMatches = matchArr.filter(m => {
                    if (!m || !m.id) return true;
                    if (seenMatch.has(m.id)) return false;
                    seenMatch.add(m.id);
                    return true;
                });

                if (uniqueMatches.length === 0) {
                    // All matches were duplicates — drop this item entirely
                    return false;
                }

                // Mutate item to only contain unique matches
                item.match = uniqueMatches;
                return true;
            });

            if (uniqueItems.length > 0) {
                deduped.push({ ...g, items: uniqueItems });
            }
        });

        if (deduped.length > 0) {
            dayGroups[day] = deduped;
        }
    });

    render(dayGroups, days);

    // Update overlay for currently playing match (scores may have changed)
    if (activeMatchId && document.getElementById('match-overlay').style.display === 'flex') {
        updateActiveMatchOverlay(dayGroups);
    }
}

function updateActiveMatchOverlay(dayGroups) {
    for (const day of Object.keys(dayGroups)) {
        for (const group of dayGroups[day]) {
            for (const item of group.items) {
                const matches = item.match || (item.competition && item.competition.match) || [];
                for (const m of matches) {
                    if (m && m.id === activeMatchId && m.home_team && m.away_team) {
                        const homeName = m.home_team.name_en || m.home_team.name || '?';
                        const awayName = m.away_team.name_en || m.away_team.name || '?';
                        const homeLogo = m.home_team.logo || '';
                        const awayLogo = m.away_team.logo || '';
                        const homeScore = (m.home_scores && m.home_scores[0] != null) ? m.home_scores[0] : '';
                        const awayScore = (m.away_scores && m.away_scores[0] != null) ? m.away_scores[0] : '';
                        const scoreStr = (homeScore !== '' && awayScore !== '') ? `${homeScore} - ${awayScore}` : '';
                        showOverlay(homeName, awayName, homeLogo, awayLogo, scoreStr);
                        return;
                    }
                }
            }
        }
    }
}

function normalizeGroups(data) {
    const groups = [];

    (data.living_group || []).forEach(g => {
        groups.push({
            title: '🔴 ' + (g.competition?.name_en || 'Live'),
            competitionName: g.competition?.name_en || '',
            items: [{ competition: g.competition, match: g.match }]
        });
    });

    (data.obs_group || []).forEach(g => {
        groups.push({
            title: g.competition?.name_en || '',
            competitionName: g.competition?.name_en || '',
            items: [{ competition: g.competition, match: g.match }]
        });
    });

    (data.hot_group || []).forEach(g => {
        groups.push({
            title: g.competition?.name_en || '',
            competitionName: g.competition?.name_en || '',
            items: [{ competition: g.competition, match: g.match }]
        });
    });

    (data.category_group || []).forEach(g => {
        groups.push({
            title: g.name,
            competitionName: g.name,
            items: g.competition_match || []
        });
    });

    (data.country_group || []).forEach(g => {
        groups.push({
            title: g.country?.name || g.name,
            competitionName: g.country?.name || g.name || '',
            items: g.competition_match || []
        });
    });

    (data.other_group || []).forEach(g => {
        groups.push({
            title: g.competition?.name_en || '',
            competitionName: g.competition?.name_en || '',
            items: [{ competition: g.competition, match: g.match }]
        });
    });

    return groups;
}

// ===== RENDER =====
// Global tooltip element (lazy created)
let tooltipEl = null;

function getTooltip() {
    if (!tooltipEl) {
        tooltipEl = document.createElement('div');
        tooltipEl.className = 'match-tooltip';
        document.body.appendChild(tooltipEl);
    }
    return tooltipEl;
}

function showTooltip(e, compName, homeName, awayName, scoreStr, matchTime, isLive) {
    const tt = getTooltip();
    let html = '';
    if (compName) {
        html += '<div class="tt-comp">' + escapeHtml(compName) + '</div>';
    }
    html += '<div class="tt-teams">' + escapeHtml(homeName) + '  vs  ' + escapeHtml(awayName) + '</div>';
    const infoParts = [];
    if (isLive) infoParts.push('🔴 LIVE');
    if (matchTime) infoParts.push('⏰ ' + matchTime);
    if (scoreStr) infoParts.push('⚽ ' + scoreStr);
    if (infoParts.length > 0) {
        html += '<div class="tt-info">' + infoParts.join('  ·  ') + '</div>';
    }
    tt.innerHTML = html;
    tt.classList.add('show');
    positionTooltip(e);
}

function positionTooltip(e) {
    const tt = getTooltip();
    const gap = 10;
    let x = e.clientX + gap;
    let y = e.clientY + gap;
    const rect = tt.getBoundingClientRect();
    if (x + rect.width > window.innerWidth) x = e.clientX - rect.width - gap;
    if (y + rect.height > window.innerHeight) y = e.clientY - rect.height - gap;
    if (x < 0) x = gap;
    if (y < 0) y = gap;
    tt.style.left = x + 'px';
    tt.style.top = y + 'px';
}

function moveTooltip(e) {
    positionTooltip(e);
}

function hideTooltip() {
    if (tooltipEl) tooltipEl.classList.remove('show');
}

function escapeHtml(str) {
    const div = document.createElement('div');
    div.appendChild(document.createTextNode(str));
    return div.innerHTML;
}

function buildMatchElement(m, homeName, awayName, homeLogo, awayLogo, scoreStr, matchTime, isLive, compName) {
    const el = document.createElement("div");
    el.className = "match";
    if (m.id === activeMatchId) el.classList.add('active-match');

    // Tooltip handlers
    el.addEventListener('mouseenter', (e) => showTooltip(e, compName, homeName, awayName, scoreStr, matchTime, isLive));
    el.addEventListener('mousemove', (e) => moveTooltip(e));
    el.addEventListener('mouseleave', hideTooltip);
    el.addEventListener('touchstart', (e) => {
        showTooltip(e.touches[0], compName, homeName, awayName, scoreStr, matchTime, isLive);
        setTimeout(hideTooltip, 2000);
    }, { passive: true });

    if (isLive) {
        const dot = document.createElement("span");
        dot.className = "match-live-dot";
        el.appendChild(dot);
    }

    if (matchTime) {
        const timeSpan = document.createElement("span");
        timeSpan.className = "match-time";
        timeSpan.innerText = matchTime;
        el.appendChild(timeSpan);
    }

    if (homeLogo) {
        const img = document.createElement("img");
        img.className = "team-logo";
        img.src = homeLogo;
        img.onerror = () => { img.style.display = 'none'; };
        el.appendChild(img);
    }

    const teamsSpan = document.createElement("span");
    teamsSpan.className = "match-teams";
    teamsSpan.innerHTML = homeName + '<span class="match-vs"> vs </span>' + awayName;
    el.appendChild(teamsSpan);

    if (awayLogo) {
        const img = document.createElement("img");
        img.className = "team-logo";
        img.src = awayLogo;
        img.onerror = () => { img.style.display = 'none'; };
        el.appendChild(img);
    }

    if (scoreStr) {
        const scoreSpan = document.createElement("span");
        scoreSpan.className = "match-score";
        scoreSpan.innerText = scoreStr;
        el.appendChild(scoreSpan);
    }

    return el;
}

function attachMatchClick(el, m, homeName, awayName, homeLogo, awayLogo, scoreStr) {
    el.onclick = async () => {
        document.querySelectorAll('.match.active-match').forEach(e => e.classList.remove('active-match'));
        el.classList.add('active-match');
        activeMatchId = m.id;

        showToast('Loading stream for ' + homeName + ' vs ' + awayName + '…', 'info');

        const r = await fetch(`/api/stream/${m.id}`);
        const s = await r.json();

        if (!s.streamUrl) {
            showToast('No stream available for ' + homeName + ' vs ' + awayName, 'warn');
            return;
        }

        play(s.streamUrl, m.id, homeName, awayName, homeLogo, awayLogo, scoreStr);

        if (isMobileDevice()) {
            closeSidebar();
        }
    };
}

function renderGroup(cat) {
    const catDiv = document.createElement("div");
    catDiv.className = "category";

    const title = document.createElement("div");
    title.className = "cat-title";
    title.innerText = cat.title || 'Unknown';

    const body = document.createElement("div");
    body.className = "cat-body";

    title.onclick = () => {
        body.style.display = body.style.display === "none" ? "block" : "none";
        title.classList.toggle('collapsed', body.style.display === "none");
    };

    const items = cat.items || [];

    items.forEach(comp => {
        if (!comp || !comp.competition) return;

        const groupTitle = document.createElement("div");
        groupTitle.className = "group-title";
        groupTitle.innerText = comp.competition.name_en || comp.competition.name || '';
        body.appendChild(groupTitle);

        const isLiveGroup = cat.title.startsWith('🔴');

        const matches = comp.match || comp.competition.match || [];
        matches.forEach(m => {
            if (!m || !m.home_team || !m.away_team) return;

            const homeName = m.home_team.name_en || m.home_team.name || '?';
            const awayName = m.away_team.name_en || m.away_team.name || '?';
            const homeLogo = m.home_team.logo || '';
            const awayLogo = m.away_team.logo || '';
            const homeScore = (m.home_scores && m.home_scores[0] != null) ? m.home_scores[0] : '';
            const awayScore = (m.away_scores && m.away_scores[0] != null) ? m.away_scores[0] : '';
            const scoreStr = (homeScore !== '' && awayScore !== '') ? `${homeScore} - ${awayScore}` : '';
            const matchTime = formatMatchTime(m.match_time);

            const compName = comp.competition.name_en || comp.competition.name || '';
            const el = buildMatchElement(m, homeName, awayName, homeLogo, awayLogo, scoreStr, matchTime, isLiveGroup, compName);
            attachMatchClick(el, m, homeName, awayName, homeLogo, awayLogo, scoreStr);
            body.appendChild(el);
        });
    });

    catDiv.appendChild(title);
    catDiv.appendChild(body);
    return catDiv;
}

function renderDateAccordion(day, groups) {
    const container = document.createElement("div");
    container.className = "category date-accordion";

    const header = document.createElement("div");
    header.className = "cat-title date-title";
    header.innerText = formatDisplayDate(day);

    const body = document.createElement("div");
    body.className = "cat-body";

    // Start expanded for today
    const todayStr = new Date().toISOString().slice(0, 10).replace(/-/g, '');
    const isToday = day === todayStr;
    if (!isToday) {
        body.style.display = "none";
        header.classList.add('collapsed');
    }

    header.onclick = () => {
        body.style.display = body.style.display === "none" ? "block" : "none";
        header.classList.toggle('collapsed', body.style.display === "none");
    };

    groups.forEach(cat => {
        const catDiv = renderGroup(cat);
        body.appendChild(catDiv);
    });

    container.appendChild(header);
    container.appendChild(body);
    return container;
}

function render(dayGroups, days) {
    const desktopList = document.getElementById("list");
    const mobileList = document.getElementById("mobile-match-list");
    desktopList.innerHTML = "";
    mobileList.innerHTML = "";

    const orderedDays = days || Object.keys(dayGroups).sort();
    let hasContent = false;

    orderedDays.forEach(day => {
        const groups = dayGroups[day];
        if (!groups || groups.length === 0) return;
        hasContent = true;

        const desktopAcc = renderDateAccordion(day, groups);
        const mobileAcc = renderDateAccordion(day, groups);
        desktopList.appendChild(desktopAcc);
        mobileList.appendChild(mobileAcc);
    });

    if (!hasContent) {
        const empty = '<div class="empty-state"><div class="empty-icon">📭</div><div>No matches found</div></div>';
        desktopList.innerHTML = empty;
        mobileList.innerHTML = empty;
    }
}

// ===== RESIZE HANDLER =====
window.addEventListener('resize', () => {
    const wasMobile = isMobile;
    isMobile = isMobileDevice();

    if (wasMobile && !isMobile) {
        document.getElementById('sidebar').classList.remove('open');
        document.getElementById('sidebar-overlay').classList.remove('show');
        document.getElementById('player-container').style.flex = '';
        document.getElementById('player-container').style.height = '';
        document.getElementById('mobile-match-list').style.display = '';
    }
});

// ===== SWIPE TO CLOSE (MOBILE) =====
let touchStartX = 0;
document.getElementById('sidebar').addEventListener('touchstart', (e) => {
    touchStartX = e.touches[0].clientX;
}, { passive: true });

document.getElementById('sidebar').addEventListener('touchmove', (e) => {
    const dx = e.touches[0].clientX - touchStartX;
    if (dx < -50) {
        closeSidebar();
    }
}, { passive: true });

// ===== INIT =====
loadMatches();
setInterval(loadMatches, 30000);
