package adminhttp

const adminPageHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>MeshChat Admin</title>
  <style>
    :root {
      color-scheme: dark;
      --bg: #08111f;
      --panel: rgba(14, 23, 41, 0.92);
      --panel-2: rgba(20, 33, 58, 0.92);
      --line: rgba(148, 163, 184, 0.18);
      --text: #e5eefb;
      --muted: #94a3b8;
      --accent: #63b3ff;
      --accent-2: #7cdbb5;
      --danger: #ff6b7a;
      --shadow: 0 24px 80px rgba(0, 0, 0, 0.35);
      font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      color: var(--text);
      background:
        radial-gradient(circle at top left, rgba(99, 179, 255, 0.18), transparent 30%),
        radial-gradient(circle at bottom right, rgba(124, 219, 181, 0.16), transparent 28%),
        linear-gradient(180deg, #09111d 0%, #060b15 100%);
    }
    .shell { max-width: 1500px; margin: 0 auto; padding: 28px; }
    .hero {
      display: flex; justify-content: space-between; gap: 24px; align-items: end;
      padding: 28px; border: 1px solid var(--line); border-radius: 24px;
      background: linear-gradient(135deg, rgba(11, 19, 34, 0.96), rgba(17, 28, 49, 0.88));
      box-shadow: var(--shadow);
    }
    .hero h1 { margin: 0 0 10px; font-size: 32px; letter-spacing: 0.02em; }
    .hero p { margin: 0; color: var(--muted); max-width: 760px; line-height: 1.6; }
    .pill {
      display: inline-flex; align-items: center; gap: 8px; padding: 8px 12px; border-radius: 999px;
      background: rgba(99, 179, 255, 0.12); color: var(--accent); border: 1px solid rgba(99, 179, 255, 0.24);
      font-size: 13px;
    }
    .grid { display: grid; gap: 18px; margin-top: 18px; grid-template-columns: 1.2fr 1.2fr; }
    .grid-three { display: grid; gap: 18px; margin-top: 18px; grid-template-columns: 1fr 1fr; }
    .card {
      border: 1px solid var(--line); border-radius: 20px; padding: 18px;
      background: linear-gradient(180deg, var(--panel), rgba(10, 16, 28, 0.96));
      box-shadow: var(--shadow);
      min-height: 120px;
    }
    .card h2 { margin: 0 0 14px; font-size: 18px; }
    .subtle { color: var(--muted); font-size: 13px; line-height: 1.5; }
    .row { display: flex; gap: 12px; flex-wrap: wrap; }
    input, textarea, select, button {
      width: 100%; border-radius: 12px; border: 1px solid rgba(148, 163, 184, 0.22);
      background: rgba(8, 15, 28, 0.84); color: var(--text); padding: 12px 14px; outline: none;
      font: inherit;
    }
    textarea { min-height: 110px; resize: vertical; }
    input::placeholder, textarea::placeholder { color: #738197; }
    input:focus, textarea:focus, select:focus { border-color: rgba(99, 179, 255, 0.7); box-shadow: 0 0 0 3px rgba(99, 179, 255, 0.12); }
    button {
      cursor: pointer; width: auto; border-color: rgba(99, 179, 255, 0.26);
      background: linear-gradient(135deg, rgba(99, 179, 255, 0.92), rgba(60, 140, 242, 0.92));
      color: #03101f; font-weight: 700;
    }
    button.secondary {
      background: rgba(15, 25, 43, 0.9); color: var(--text);
    }
    button.danger {
      background: linear-gradient(135deg, rgba(255, 107, 122, 0.96), rgba(233, 76, 97, 0.96));
      color: #22070b;
    }
    button.ghost {
      background: transparent;
    }
    .login {
      margin-top: 18px; max-width: 480px;
    }
    .stack { display: grid; gap: 12px; }
    .toolbar { display: flex; gap: 10px; flex-wrap: wrap; margin-bottom: 12px; }
    .table-wrap { overflow: auto; border: 1px solid rgba(148, 163, 184, 0.14); border-radius: 16px; }
    table { width: 100%; border-collapse: collapse; min-width: 620px; }
    th, td { padding: 11px 12px; border-bottom: 1px solid rgba(148, 163, 184, 0.12); text-align: left; vertical-align: top; }
    th { position: sticky; top: 0; background: #0f1a2f; font-size: 12px; text-transform: uppercase; letter-spacing: 0.06em; color: #aac0de; }
    tr:hover td { background: rgba(99, 179, 255, 0.05); }
    .muted { color: var(--muted); }
    .status { padding: 10px 14px; border-radius: 14px; background: rgba(124, 219, 181, 0.1); border: 1px solid rgba(124, 219, 181, 0.2); }
    .split { display: grid; gap: 18px; grid-template-columns: 1fr 1fr; margin-top: 18px; }
    pre {
      margin: 0; white-space: pre-wrap; word-break: break-word;
      background: rgba(6, 12, 23, 0.92); border: 1px solid rgba(148, 163, 184, 0.14);
      border-radius: 16px; padding: 14px; max-height: 380px; overflow: auto;
    }
    .hidden { display: none !important; }
    .mini { font-size: 12px; color: var(--muted); }
    .group-card { display: flex; gap: 12px; flex-wrap: wrap; align-items: center; justify-content: space-between; }
    .group-actions { display: flex; gap: 8px; flex-wrap: wrap; }
    @media (max-width: 1100px) {
      .grid, .grid-three, .split { grid-template-columns: 1fr; }
      .hero { align-items: start; flex-direction: column; }
    }
  </style>
</head>
<body>
  <div class="shell">
    <div class="hero">
      <div>
        <div class="pill">MeshChat Admin Panel</div>
        <h1>后台管理控制台</h1>
        <p>用于浏览器直连管理服务器。可以登录、查看用户和群列表、创建和修改群聊、解散群聊、查看成员和聊天记录。</p>
      </div>
      <div class="status" id="sessionStatus">未登录</div>
    </div>

    <div class="card login" id="loginCard">
      <h2>管理员登录</h2>
      <div class="stack">
        <input id="loginUsername" placeholder="管理员用户名" value="admin">
        <input id="loginPassword" placeholder="管理员密码" type="password" value="admin123456">
        <button onclick="login()">登录后台</button>
      </div>
      <div class="mini" style="margin-top:10px;">登录后会把 token 存在浏览器 localStorage。</div>
    </div>

    <div id="dashboard" class="hidden">
      <div class="toolbar" style="margin-top:18px;">
        <button onclick="loadMe()">刷新登录信息</button>
        <button class="secondary" onclick="loadUsers()">刷新用户</button>
        <button class="secondary" onclick="loadGroups()">刷新群聊</button>
        <button class="ghost" onclick="logout()">退出登录</button>
      </div>

      <div class="grid-three">
        <div class="card">
          <h2>用户列表</h2>
          <div class="subtle">展示服务器上的用户记录。</div>
          <div class="table-wrap" style="margin-top:12px;">
            <table>
              <thead><tr><th>ID</th><th>用户名</th><th>昵称</th><th>状态</th></tr></thead>
              <tbody id="usersBody"><tr><td colspan="4" class="muted">暂无数据</td></tr></tbody>
            </table>
          </div>
        </div>

        <div class="card">
          <h2>群聊列表</h2>
          <div class="subtle">点击某一行后可在右侧查看详情、成员和聊天记录。</div>
          <div class="table-wrap" style="margin-top:12px;">
            <table>
              <thead><tr><th>群ID</th><th>标题</th><th>状态</th><th>操作</th></tr></thead>
              <tbody id="groupsBody"><tr><td colspan="4" class="muted">暂无数据</td></tr></tbody>
            </table>
          </div>
        </div>

      </div>

      <div class="grid">
        <div class="card">
          <h2>创建群聊</h2>
          <div class="stack">
            <input id="createTitle" placeholder="标题">
            <textarea id="createAbout" placeholder="介绍"></textarea>
            <input id="createAvatarCID" placeholder="头像 CID">
            <select id="createMemberListVisibility">
              <option value="visible">visible</option>
              <option value="hidden">hidden</option>
            </select>
            <select id="createJoinMode">
              <option value="invite_only">invite_only</option>
              <option value="open">open</option>
            </select>
            <input id="createDefaultPermissions" placeholder="default_permissions" value="2047">
            <button onclick="createGroup()">创建</button>
          </div>
        </div>
        <div class="card">
          <h2>创建说明</h2>
          <div class="subtle">这里放创建群聊时的配置建议。公开群请把 <code>join_mode</code> 设成 <code>open</code>，普通群保持 <code>invite_only</code>。群创建后可以再在右侧修改资料和状态。</div>
        </div>
      </div>

      <div class="grid">
        <div class="card">
          <h2>常用配置</h2>
          <div class="stack">
            <div class="status"><code>member_list_visibility=visible</code> 适合公开群</div>
            <div class="status"><code>member_list_visibility=hidden</code> 适合隐私群</div>
            <div class="status"><code>default_permissions=2047</code> 是默认权限集</div>
          </div>
        </div>
        <div class="card">
          <h2>提示</h2>
          <div class="subtle">创建后可立即在群聊列表里点击查看，并进入成员和聊天记录面板。</div>
        </div>
      </div>

      <div class="split">
        <div class="card">
          <div class="group-card">
            <div>
              <h2 style="margin-bottom:6px;">群聊详情</h2>
              <div class="subtle" id="selectedGroupHint">尚未选择群聊</div>
            </div>
            <div class="group-actions">
              <button class="secondary" onclick="saveGroup()">保存修改</button>
              <button class="danger" onclick="dissolveGroup()">解散群聊</button>
            </div>
          </div>
          <div class="stack" style="margin-top:12px;">
            <input id="editTitle" placeholder="标题">
            <textarea id="editAbout" placeholder="介绍"></textarea>
            <input id="editAvatarCID" placeholder="头像 CID">
            <select id="editMemberListVisibility">
              <option value="visible">visible</option>
              <option value="hidden">hidden</option>
            </select>
            <select id="editJoinMode">
              <option value="invite_only">invite_only</option>
              <option value="open">open</option>
            </select>
            <select id="editStatus">
              <option value="active">active</option>
              <option value="closed">closed</option>
            </select>
          </div>
        </div>

        <div class="card">
          <h2>成员 / 聊天记录</h2>
          <div class="subtle" id="selectedGroupMeta">请选择左侧群聊。</div>
          <div class="toolbar" style="margin-top:12px;">
            <button class="secondary" onclick="loadMembers()">刷新成员</button>
            <button class="secondary" onclick="loadMessages()">刷新聊天</button>
          </div>
          <div class="toolbar" style="margin-top:12px; align-items:center; gap:8px;">
            <input id="memberTargetUserId" placeholder="成员 user_id" style="max-width:180px;">
            <button class="secondary" onclick="setMemberAdmin(true)">设为管理员</button>
            <button class="secondary" onclick="setMemberAdmin(false)">取消管理员</button>
            <input id="ownerTargetUserId" placeholder="新群主 user_id" style="max-width:180px;">
            <button onclick="transferOwner()">转让群主</button>
          </div>
          <div class="stack">
            <div>
              <div class="mini">成员列表</div>
              <pre id="membersView">[]</pre>
            </div>
            <div>
              <div class="mini">聊天记录</div>
              <pre id="messagesView">[]</pre>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>

  <script>
    const tokenKey = 'meshchat_admin_token';
    const apiBase = '';
    let currentGroup = null;

    function token() { return localStorage.getItem(tokenKey) || ''; }
    function setToken(value) { localStorage.setItem(tokenKey, value); }
    function clearToken() { localStorage.removeItem(tokenKey); }

    function setSession(status) {
      document.getElementById('sessionStatus').textContent = status;
    }

    function showDashboard() {
      document.getElementById('loginCard').classList.add('hidden');
      document.getElementById('dashboard').classList.remove('hidden');
    }

    function showLogin() {
      document.getElementById('loginCard').classList.remove('hidden');
      document.getElementById('dashboard').classList.add('hidden');
    }

    async function api(path, options = {}) {
      const headers = Object.assign({
        'Content-Type': 'application/json'
      }, options.headers || {});
      const t = token();
      if (t) headers['Authorization'] = 'Bearer ' + t;
      const resp = await fetch(apiBase + path, Object.assign({}, options, { headers }));
      const text = await resp.text();
      let data = null;
      if (text) {
        try { data = JSON.parse(text); } catch (e) { data = text; }
      }
      if (!resp.ok) {
        const message = data && data.error && data.error.message ? data.error.message : ('HTTP ' + resp.status);
        throw new Error(message);
      }
      return data;
    }

    async function login() {
      try {
        const data = await api('/admin/login', {
          method: 'POST',
          body: JSON.stringify({
            username: document.getElementById('loginUsername').value.trim(),
            password: document.getElementById('loginPassword').value
          }),
          headers: {}
        });
        setToken(data.token);
        setSession('已登录: ' + data.username);
        showDashboard();
        await bootstrap();
      } catch (err) {
        alert(err.message);
      }
    }

    async function loadMe() {
      try {
        const data = await api('/admin/me');
        setSession('已登录: ' + data.username);
      } catch (err) {
        handleAuthError(err);
      }
    }

    async function loadUsers() {
      try {
        const data = await api('/admin/users?limit=100');
        const body = document.getElementById('usersBody');
        if (!data.length) {
          body.innerHTML = '<tr><td colspan="4" class="muted">暂无数据</td></tr>';
          return;
        }
        body.innerHTML = data.map(u => '<tr><td>' + u.id + '</td><td>' + escapeHtml(u.username) + '</td><td>' + escapeHtml(u.display_name || '') + '</td><td>' + escapeHtml(u.status || '') + '</td></tr>').join('');
      } catch (err) {
        handleAuthError(err);
      }
    }

    async function loadGroups() {
      try {
        const data = await api('/admin/groups?limit=100');
        const body = document.getElementById('groupsBody');
        if (!data.length) {
          body.innerHTML = '<tr><td colspan="4" class="muted">暂无数据</td></tr>';
          return;
        }
        body.innerHTML = data.map(g => '<tr>' +
          '<td>' + escapeHtml(g.group_id) + '</td>' +
          '<td><a href="#" onclick="selectGroup(\'' + escapeAttr(g.group_id) + '\');return false;">' + escapeHtml(g.title || '') + '</a></td>' +
          '<td>' + escapeHtml(g.status || '') + '</td>' +
          '<td><button class="secondary" onclick="selectGroup(\'' + escapeAttr(g.group_id) + '\')">查看</button></td>' +
          '</tr>').join('');
      } catch (err) {
        handleAuthError(err);
      }
    }

    async function createGroup() {
      try {
        const payload = {
          title: document.getElementById('createTitle').value.trim(),
          about: document.getElementById('createAbout').value.trim(),
          avatar_cid: document.getElementById('createAvatarCID').value.trim(),
          member_list_visibility: document.getElementById('createMemberListVisibility').value,
          join_mode: document.getElementById('createJoinMode').value,
          default_permissions: Number(document.getElementById('createDefaultPermissions').value || 0)
        };
        await api('/admin/groups', { method: 'POST', body: JSON.stringify(payload) });
        await loadGroups();
        alert('创建成功');
      } catch (err) {
        handleAuthError(err);
      }
    }

    async function selectGroup(groupID) {
      try {
        const group = await api('/admin/groups/' + encodeURIComponent(groupID));
        currentGroup = group;
        document.getElementById('selectedGroupHint').textContent = '当前群: ' + group.group_id;
        document.getElementById('selectedGroupMeta').textContent = group.title + ' / ' + group.status;
        document.getElementById('memberTargetUserId').value = '';
        document.getElementById('ownerTargetUserId').value = group.owner_user && group.owner_user.id ? group.owner_user.id : '';
        document.getElementById('editTitle').value = group.title || '';
        document.getElementById('editAbout').value = group.about || '';
        document.getElementById('editAvatarCID').value = group.avatar_cid || '';
        document.getElementById('editMemberListVisibility').value = group.member_list_visibility || 'visible';
        document.getElementById('editJoinMode').value = group.join_mode || 'invite_only';
        document.getElementById('editStatus').value = group.status || 'active';
        await loadMembers();
        await loadMessages();
      } catch (err) {
        handleAuthError(err);
      }
    }

    async function saveGroup() {
      if (!currentGroup) {
        alert('请先选择一个群聊');
        return;
      }
      try {
        const payload = {
          title: document.getElementById('editTitle').value.trim(),
          about: document.getElementById('editAbout').value.trim(),
          avatar_cid: document.getElementById('editAvatarCID').value.trim(),
          member_list_visibility: document.getElementById('editMemberListVisibility').value,
          join_mode: document.getElementById('editJoinMode').value,
          status: document.getElementById('editStatus').value
        };
        await api('/admin/groups/' + encodeURIComponent(currentGroup.group_id), {
          method: 'PATCH',
          body: JSON.stringify(payload)
        });
        await loadGroups();
        await selectGroup(currentGroup.group_id);
        alert('保存成功');
      } catch (err) {
        handleAuthError(err);
      }
    }

    async function dissolveGroup() {
      if (!currentGroup) {
        alert('请先选择一个群聊');
        return;
      }
      if (!confirm('确认解散该群聊？')) return;
      try {
        await api('/admin/groups/' + encodeURIComponent(currentGroup.group_id) + '/dissolve', { method: 'POST' });
        currentGroup = null;
        document.getElementById('selectedGroupHint').textContent = '尚未选择群聊';
        document.getElementById('selectedGroupMeta').textContent = '请选择左侧群聊。';
        document.getElementById('membersView').textContent = '[]';
        document.getElementById('messagesView').textContent = '[]';
        await loadGroups();
        alert('已解散');
      } catch (err) {
        handleAuthError(err);
      }
    }

    async function loadMembers() {
      if (!currentGroup) return;
      try {
        const data = await api('/admin/groups/' + encodeURIComponent(currentGroup.group_id) + '/members');
        document.getElementById('membersView').textContent = JSON.stringify(data, null, 2);
      } catch (err) {
        handleAuthError(err);
      }
    }

    async function setMemberAdmin(isAdmin) {
      if (!currentGroup) {
        alert('请先选择一个群聊');
        return;
      }
      const userID = Number(document.getElementById('memberTargetUserId').value || 0);
      if (!userID) {
        alert('请输入成员 user_id');
        return;
      }
      try {
        await api('/admin/groups/' + encodeURIComponent(currentGroup.group_id) + '/members/' + encodeURIComponent(String(userID)) + '/admin', {
          method: 'POST',
          body: JSON.stringify({ is_admin: isAdmin })
        });
        await loadMembers();
        alert(isAdmin ? '已设为管理员' : '已取消管理员');
      } catch (err) {
        handleAuthError(err);
      }
    }

    async function transferOwner() {
      if (!currentGroup) {
        alert('请先选择一个群聊');
        return;
      }
      const userID = Number(document.getElementById('ownerTargetUserId').value || 0);
      if (!userID) {
        alert('请输入新群主 user_id');
        return;
      }
      if (!confirm('确认把群主转让给 user_id=' + userID + ' ?')) return;
      try {
        await api('/admin/groups/' + encodeURIComponent(currentGroup.group_id) + '/transfer-owner', {
          method: 'POST',
          body: JSON.stringify({ user_id: userID })
        });
        await selectGroup(currentGroup.group_id);
        await loadGroups();
        alert('已转让群主');
      } catch (err) {
        handleAuthError(err);
      }
    }

    async function loadMessages() {
      if (!currentGroup) return;
      try {
        const data = await api('/admin/groups/' + encodeURIComponent(currentGroup.group_id) + '/messages?limit=50');
        document.getElementById('messagesView').textContent = JSON.stringify(data, null, 2);
      } catch (err) {
        handleAuthError(err);
      }
    }

    function logout() {
      clearToken();
      currentGroup = null;
      setSession('未登录');
      showLogin();
    }

    function handleAuthError(err) {
      if ((err && err.message || '').toLowerCase().includes('authorization') || (err && err.message || '').toLowerCase().includes('token')) {
        logout();
      } else {
        alert(err.message);
      }
    }

    async function bootstrap() {
      if (!token()) {
        showLogin();
        return;
      }
      showDashboard();
      await loadMe();
      await loadUsers();
      await loadGroups();
    }

    function escapeHtml(value) {
      return String(value || '')
        .replaceAll('&', '&amp;')
        .replaceAll('<', '&lt;')
        .replaceAll('>', '&gt;')
        .replaceAll('"', '&quot;')
        .replaceAll("'", '&#39;');
    }

    function escapeAttr(value) {
      return String(value || '').replaceAll("'", '&#39;');
    }

    bootstrap();
  </script>
</body>
</html>`
