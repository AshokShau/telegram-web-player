package bot

import (
	"ashokshau/tg-web/internal/config"
	"ashokshau/tg-web/internal/utils"
	"fmt"
	"html"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

func helpHandler(c *td.Client, m *td.Message) error {
	if !m.IsPrivate() {
		return startHandler(c, m)
	}
	if err := sendMenuWidget(c, m, "help"); err == nil {
		return nil
	} else {
		c.Logger.Warnf("help widget unavailable, using rich message: %v", err)
	}
	_, err := m.ReplyRichMessageHTML(c,
		"<h3>Command guide</h3><p>Choose a category below to browse commands.</p>",
		&td.SendRichTextOpts{SendMessageOpts: td.SendMessageOpts{ReplyMarkup: utils.HelpMenuKeyboard()}},
	)
	return err
}

func sendMenuWidget(c *td.Client, m *td.Message, view string) error {
	username := c.Me().Usernames.EditableUsername
	if username == "" {
		return fmt.Errorf("bot username is unavailable")
	}
	name := html.EscapeString(c.Me().FirstName)
	if name == "" {
		name = "SyncTune"
	}
	viewer := html.EscapeString(firstName(c, m))
	if viewer == "" {
		viewer = "there"
	}
	widget := strings.NewReplacer(
		"{{BOT}}", name,
		"{{VIEWER}}", viewer,
		"{{VIEW}}", view,
		"{{ADD_URL}}", "https://t.me/"+username+"?startgroup=true",
		"{{SUPPORT_URL}}", html.EscapeString(config.SupportGroup),
	).Replace(menuWidgetHTML)
	caption := "SyncTune · Music together in Telegram. Use /help for commands and /privacy for data use."
	_, err := m.ReplyWidget(c, widget, &td.SendWidgetOpts{
		Caption:      caption,
		Width:        0,
		Height:       0,
		AllowQueries: true,
	})
	return err
}

const menuFAQData = `<details class="faq-item"><summary class="faq-question">How do I start listening?</summary><p class="faq-answer">Add the bot to a group, send /play with a song or link, then use /player to join the room.</p></details><details class="faq-item"><summary class="faq-question">What music can I play?</summary><p class="faq-answer">Search for songs or use supported YouTube, Spotify, SoundCloud, and Telegram audio links.</p></details><details class="faq-item"><summary class="faq-question">Who can control playback?</summary><p class="faq-answer">The chat owner sets play permissions with /settings. Some controls require admin or authorized access.</p></details><details class="faq-item"><summary class="faq-question">Where can I read about privacy?</summary><p class="faq-answer">Send /privacy for data use, room visibility, and deletion requests.</p></details>`

func menuWidgetQueryHandler(c *td.Client, q *td.UpdateNewWidgetQuery) error {
	return q.Answer(c, menuFAQData)
}

const menuWidgetHTML = `<!doctype html><html><head><meta name="viewport" content="width=device-width,initial-scale=1"><style>
:root{color-scheme:dark light;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;font-size:14px}
*{box-sizing:border-box}html,body{width:100%;max-width:100%;margin:0}body{padding:8px;background:#20242c;color:#f3f5f9;line-height:1.35;overflow:hidden}
.shell{width:100%;min-width:0;padding:14px;border:1px solid #3a414d;border-radius:18px;background:#272c35}
.head{display:flex;align-items:center;gap:10px;min-width:0;margin-bottom:12px}.head>div:last-child{min-width:0}.mark{width:38px;height:38px;flex:none;border-radius:12px;background:#1683ce;color:#fff;display:grid;place-items:center;font-size:19px}.title{overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-weight:700;font-size:16px;letter-spacing:-.02em}.sub,.hint,.eyebrow{color:#b0bac9;font-size:12px}.sub{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.hint,.eyebrow{letter-spacing:.07em;text-transform:uppercase;font-weight:700}.hero{font-size:19px;font-weight:700;letter-spacing:-.03em;line-height:1.18;margin:5px 0 6px}.lead{color:#b0bac9;font-size:13px;line-height:1.4;margin:0 0 12px}.eyebrow{margin:0 0 5px}
.tabs{display:flex;gap:4px;background:#20242c;border-radius:12px;padding:4px;margin-bottom:14px}.tab{flex:1;min-width:0;min-height:36px;background:transparent;border:0;border-radius:10px;color:#b0bac9;font-family:inherit;font-size:13px;font-weight:650}.tab.active{background:#30465b;color:#79c5ff}
.grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:8px}.tile,.action,.category{border:1px solid #3a414d;background:#303640;color:#f3f5f9;border-radius:12px;font-family:inherit}.tile{display:block;width:100%;min-width:0;min-height:90px;padding:9px;text-align:left;overflow:hidden}.icon{width:29px;height:29px;display:grid;place-items:center;margin-bottom:7px;border-radius:9px;background:#30465b;color:#79c5ff;font-size:17px}.tile:nth-child(2) .icon{background:#22453f;color:#8ce6c2}.tile:nth-child(3) .icon{background:#443758;color:#d4b2ff}.tile:nth-child(4) .icon{background:#504032;color:#ffd097}.tile strong,.tile small{display:block;overflow-wrap:anywhere}.tile strong{font-size:12px;line-height:1.2}.tile small{font-size:11px;color:#b0bac9;margin-top:2px;line-height:1.2}
.action{display:grid;place-items:center;min-width:0;min-height:42px;text-align:center;text-decoration:none;padding:8px;margin-top:8px;font-size:12px;font-weight:650;line-height:1.2}.primary{background:#1683ce;border-color:#1683ce;color:#fff;font-size:13px}.actions{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:8px}
#categories{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:8px}.category{display:block;width:100%;min-width:0;min-height:72px;padding:9px;text-align:left;overflow:hidden}.cat-icon{width:28px;height:28px;display:grid;place-items:center;margin-bottom:7px;border-radius:9px;background:#1683ce;color:#fff;font-size:15px}.category:nth-child(2) .cat-icon{background:#299346}.category:nth-child(3) .cat-icon{background:#9950d1}.category:nth-child(4) .cat-icon{background:#ce7916}.category:nth-child(5) .cat-icon{background:#078c9b}.category:nth-child(6) .cat-icon{background:#d74d65}.category strong{display:block;font-size:12px;line-height:1.2}.commands{display:none}.commands.active{display:block}.row{display:flex;width:100%;gap:8px;padding:9px 0;text-align:left;background:transparent;color:inherit;border:0;border-bottom:1px solid #3a414d;font-family:inherit;cursor:pointer}.row:last-child{border-bottom:0}.row:active{background:#30465b}.copy-hint{font-size:11px;color:#b0bac9;margin:0 0 4px}.cmd{color:#79c5ff;font-weight:700;min-width:0;width:39%;flex:none;font-size:11px;overflow-wrap:anywhere}.desc{color:#f3f5f9;font-size:11px;line-height:1.25;min-width:0}.back{background:transparent;color:#79c5ff;border:0;padding:5px 0 9px;font-family:inherit;font-size:12px;font-weight:650}.faq-item{background:#303640;border:1px solid #3a414d;border-radius:12px;margin:7px 0}.faq-question{display:block;width:100%;padding:11px;text-align:left;color:inherit;font:inherit;font-weight:650;cursor:pointer}.faq-question:before{content:"▸";color:#79c5ff;margin-right:8px}.faq-item[open] .faq-question:before{content:"▾"}.faq-answer{display:none;margin:0;padding:0 11px 11px;color:#b0bac9;font-size:12px;line-height:1.4}.faq-item[open] .faq-answer{display:block}.panel{display:none}.panel.active{display:block}.panel.detail>.eyebrow,.panel.detail>.hero,.panel.detail>.lead,.panel.detail>.actions{display:none}button{cursor:pointer}button:active,.action:active{opacity:.8}@media(max-width:280px){#categories{grid-template-columns:1fr}.category{min-height:60px}.cat-icon{float:left;margin:0 9px 0 0}}
@media(prefers-color-scheme:light){body{background:#e8edf3;color:#202e40}.shell{background:#eef2f7;border-color:#cbd5e2}.sub,.hint,.eyebrow,.lead,.tile small{color:#536278}.tabs{background:#e8edf3}.tab{color:#536278}.tab.active{background:#d2e4f3;color:#1261a0}.tile,.action,.category,.faq-item{background:#f5f7fa;border-color:#cbd5e2;color:#202e40}.faq-answer{color:#536278}.faq-question:before{color:#1261a0}.primary{background:#1676b9;border-color:#1676b9;color:#fff}.icon{background:#d2e4f3;color:#1261a0}.row{border-color:#cbd5e2}.cmd,.back{color:#1261a0}.desc{color:#34445a}}
</style></head><body><main class="shell"><header class="head"><div class="mark">♫</div><div><div class="title">{{BOT}}</div><div class="sub">Music together, right in Telegram</div></div></header><nav class="tabs"><button class="tab" data-tab="home">Start</button><button class="tab" data-tab="help">Commands</button><button class="tab" data-tab="faq">FAQ</button></nav>
<section class="panel" id="home"><div class="hint">HELLO, {{VIEWER}}</div><h1 class="hero">Your group, your soundtrack.</h1><p class="lead">Add the bot to a group, share a song, and listen in sync in the Web Player.</p><div class="grid"><button class="tile" data-open-category="listen"><span class="icon">▶</span><span><strong>Play anything</strong><small>/play song or link</small></span></button><button class="tile" data-open-category="queue"><span class="icon">▤</span><span><strong>Queue together</strong><small>/queue to see what's next</small></span></button><button class="tile" data-open-category="general"><span class="icon">◎</span><span><strong>Open the room</strong><small>/player in your group</small></span></button><button class="tile" data-open-category="playlist"><span class="icon">♡</span><span><strong>Keep favorites</strong><small>Personal playlists</small></span></button></div><a class="action primary" href="{{ADD_URL}}">＋ Add to a group</a><div class="actions"><button class="action" data-tab="help">Explore commands</button><a class="action" href="{{SUPPORT_URL}}">Get support</a></div></section>
<section class="panel" id="help"><div class="eyebrow">COMMAND GUIDE</div><h1 class="hero">Find your next move.</h1><p class="lead">Tap a section for commands and what they do.</p><div id="categories"><button class="category" data-category="listen"><span class="cat-icon" aria-hidden="true">▶</span><strong>Play &amp; control</strong></button><button class="category" data-category="queue"><span class="cat-icon" aria-hidden="true">▤</span><strong>Queue</strong></button><button class="category" data-category="playlist"><span class="cat-icon" aria-hidden="true">♡</span><strong>Your playlists</strong></button><button class="category" data-category="room"><span class="cat-icon" aria-hidden="true">⚙</span><strong>Room &amp; access</strong></button><button class="category" data-category="general"><span class="cat-icon" aria-hidden="true">◎</span><strong>General &amp; support</strong></button><button class="category" data-category="developer"><span class="cat-icon" aria-hidden="true">⌘</span><strong>Developer</strong></button></div><div class="commands" id="command-list"><button class="back" id="back">← All sections</button><div class="eyebrow" id="section-title"></div><p class="copy-hint" id="copy-hint">Tap a command to copy it</p><div id="rows"></div></div></section><section class="panel" id="faq"><h1 class="hero">Quick answers.</h1><div id="faq-list" class="faq-list"></div></section></main><script>
const groups={listen:{title:'Play & control',items:[['/play song','Play a song or link.'],['/fplay song','Play next, ahead of the queue.'],['/pause','Pause playback.'],['/resume','Resume playback.'],['/seek seconds','Jump to a position.'],['/stop','Stop and clear the room.']]},queue:{title:'Queue',items:[['/queue','See upcoming tracks.'],['/skip','Skip the current track.'],['/remove index','Remove a queued track.'],['/loop 0–10','Repeat the current track.']]},playlist:{title:'Your playlists',items:[['/createplaylist','Create a playlist.'],['/deleteplaylist','Delete a playlist.'],['/addtoplaylist','Save a track.'],['/removefromplaylist','Remove a saved track.'],['/playlistinfo','View playlist details.'],['/myplaylists','List your playlists.']]},room:{title:'Room & access',items:[['/settings','Manage room settings; chat owner.'],['/auth','Authorize a user; admin.'],['/removeAuth','Remove authorization; admin.'],['/auths','List authorized users; admin.'],['/reload','Refresh admin cache; admin.']]},general:{title:'General & support',items:[['/player','Open the Web Player.'],['/start','Show the welcome card.'],['/help','Show this guide.'],['/ping','Check bot response and uptime.'],['/privacy','Read data use and deletion details.']]},developer:{title:'Developer',items:[['/stats','Bot and hosting statistics.'],['/av','Active voice and video chats.'],['/logger','Logging configuration.']]}};
let faqLoaded=false;function loadFAQ(){if(faqLoaded)return;const box=document.getElementById('faq-list');box.textContent='Loading answers…';if(typeof TelegramWidget.request!=='function'){box.textContent='FAQ needs a newer Telegram client.';return}TelegramWidget.request('menu:faq').then(a=>{if(!a.result)throw Error('empty FAQ');box.innerHTML=a.result;faqLoaded=true}).catch(()=>box.textContent='Answers unavailable. Try again, or use Start → Get support.')}function tab(name){document.querySelectorAll('.panel').forEach(e=>e.classList.toggle('active',e.id===name));document.querySelectorAll('.tab').forEach(e=>e.classList.toggle('active',e.dataset.tab===name));if(name==='faq')loadFAQ()}document.querySelectorAll('[data-tab]').forEach(e=>e.addEventListener('click',()=>tab(e.dataset.tab)));
document.querySelectorAll('[data-open-category]').forEach(e=>e.addEventListener('click',()=>{tab('help');document.querySelector('[data-category="'+e.dataset.openCategory+'"]').click()}));document.querySelectorAll('[data-category]').forEach(e=>e.addEventListener('click',()=>{const g=groups[e.dataset.category];document.getElementById('section-title').textContent=g.title;const rows=document.getElementById('rows');rows.textContent='';g.items.forEach(([command,description])=>{const row=document.createElement('button');row.type='button';row.className='row';row.setAttribute('aria-label','Copy '+command.split(' ')[0]);const a=document.createElement('span');a.className='cmd';a.textContent=command+'  ⧉';const b=document.createElement('span');b.className='desc';b.textContent=description;row.append(a,b);row.addEventListener('click',()=>{if(typeof TelegramWidget.copyText==='function')TelegramWidget.copyText(command.split(' ')[0])});rows.append(row)});document.getElementById('categories').style.display='none';document.getElementById('command-list').classList.add('active');document.getElementById('help').classList.add('detail')}));document.getElementById('back').addEventListener('click',()=>{document.getElementById('categories').style.display='';document.getElementById('command-list').classList.remove('active');document.getElementById('help').classList.remove('detail')});tab('{{VIEW}}');if(typeof TelegramWidget.copyText!=='function')document.getElementById('copy-hint').textContent='Copy requires a newer Telegram client';TelegramWidget.ready();
</script></body></html>`
