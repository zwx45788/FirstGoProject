class ChatClient {
  constructor() {
    this.ws = null;
    this.currentMode = 'public';
    this.currentUsername = '';
    this.selectedUser = null;
    this.api = this.createAPI();
    this.init();
  }

  init() {
    this.setupEventListeners();
    this.connect();
  }

  connect() {
    this.ws = new WebSocket('ws://localhost:8081/ws');

    this.ws.onopen = () => {
      // 连接成功后，直接弹出昵称输入框
      this.askForUsername();
    };

    this.ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        this.handleServerMessage(data);
      } catch {
        this.addSystemMessage(`⚠️ 无法解析服务器消息: ${event.data}`);
      }
    };

    this.ws.onclose = () => {
      this.addSystemMessage('❌ 与服务器断开连接');
      document.getElementById('connectionStatus').textContent = '连接断开';
    };

    this.ws.onerror = (err) => {
      console.error('WebSocket错误:', err);
      this.addSystemMessage('❗ 连接出错');
    };
  }

  // --- 修改：简化为直接询问昵称 ---
  askForUsername() {
    // 直接显示欢迎模态框
    //console.log('askForUsername 被调用了'); // 添加这行
    document.getElementById('welcomeModal').style.display = 'flex';
    // 聚焦到输入框
    document.getElementById('welcomeUsernameInput').focus();
  }
  setWelcomeUsername() {
    const input = document.getElementById('welcomeUsernameInput');
    const username = input.value.trim();

    if (username) {
      // 不再保存到localStorage
      // localStorage.setItem('chatUsername', username); 

      this.currentUsername = username;
      document.getElementById('currentUsername').textContent = this.currentUsername;
      
      // 关闭模态框
      document.getElementById('welcomeModal').style.display = 'none';
      
      // 发送改名请求给服务器
      this.api.rename(username);
      
      // 清空输入框
      input.value = '';
      
      this.addSystemMessage(`✅ 你的昵称已设置为 ${username}`);
      document.getElementById('connectionStatus').textContent = '已连接';
    } else {
      // 如果用户没输入，给个提示
      input.style.borderColor = 'red';
      input.placeholder = '昵称不能为空！';
    }
  }
  

  // --- 高层统一接口 ---
  createAPI() {
    return {
      sendPublic: (msg) => this.ws.send(JSON.stringify({ type: 'chat', content: msg })),
      sendPrivate: (to, msg) => this.ws.send(JSON.stringify({ type: 'private', to, content: msg })),
      requestUserList: () => this.ws.send(JSON.stringify({ type: 'who' })),
      rename: (newName) => this.ws.send(JSON.stringify({ type: 'rename', newName })),
    };
  }

  // --- 后端消息处理路由 ---
  handleServerMessage(msg) {
    switch (msg.type) {
      case 'system':
        this.addSystemMessage(msg.content);
        break;

      case 'chat':
        this.addChatMessage(msg.from, msg.content, msg.from === this.currentUsername);
        break;

      case 'private':
        this.addChatMessage(
          msg.from,
          `(私聊) ${msg.content}`,
          msg.from === this.currentUsername
        );
        break;

      case 'user_list':
        this.updateUsersList(msg.users);
        break;

      case 'rename':
        this.currentUsername = msg.newName;
        document.getElementById('currentUsername').textContent = this.currentUsername;
        this.addSystemMessage(`你的昵称已改为 ${msg.newName}`);
        break;

      default:
        console.warn('未知消息类型:', msg);
        this.addSystemMessage(`收到未知消息: ${JSON.stringify(msg)}`);
    }
  }

  // --- UI渲染逻辑 ---
  addChatMessage(from, content, isOwn = false) {
    const messagesDiv = document.getElementById('messages');
    const msgDiv = document.createElement('div');
    msgDiv.className = `message ${isOwn ? 'own' : 'other'}`;

    const time = new Date().toLocaleTimeString();
    msgDiv.innerHTML = `
      <div class="message-info">${from} ${time}</div>
      <div>${content}</div>
    `;

    messagesDiv.appendChild(msgDiv);
    messagesDiv.scrollTop = messagesDiv.scrollHeight;
  }

  addSystemMessage(content) {
    const messagesDiv = document.getElementById('messages');
    const sysDiv = document.createElement('div');
    sysDiv.className = 'system-message';
    sysDiv.textContent = `[系统] ${content}`;
    messagesDiv.appendChild(sysDiv);
    messagesDiv.scrollTop = messagesDiv.scrollHeight;
  }

 updateUsersList(users) {
  const container = document.getElementById('usersContainer');
  container.innerHTML = '';

  // --- 关键修改在这里 ---
  // 从服务器返回的用户列表中，过滤掉当前用户自己
  const otherUsers = users.filter(user => user !== this.currentUsername);

  // 遍历过滤后的用户列表
  otherUsers.forEach(user => {
    const div = document.createElement('div');
    div.className = 'user-item';
    div.textContent = user;
    div.onclick = () => this.selectUser(user);
    container.appendChild(div);
  });
}

  selectUser(username) {
    this.selectedUser = username;
    this.currentMode = 'private';
    document.getElementById('chatHeader').textContent = `与 ${username} 私聊`;
    this.addSystemMessage(`开始与 ${username} 私聊`);
  }

  sendMessage() {
    const input = document.getElementById('messageInput');
    const content = input.value.trim();
    if (!content) return;

    if (this.currentMode === 'public') {
      this.api.sendPublic(content);
    } else if (this.currentMode === 'private' && this.selectedUser) {
      this.api.sendPrivate(this.selectedUser, content);
    }

    input.value = '';
  }

  setupEventListeners() {
    document.querySelectorAll('.menu-item').forEach(item => {
      item.addEventListener('click', (e) => {
        document.querySelectorAll('.menu-item').forEach(i => i.classList.remove('active'));
        e.target.classList.add('active');
        this.switchMode(e.target.dataset.mode);
      });
    });

    document.getElementById('sendBtn').addEventListener('click', () => this.sendMessage());
    document.getElementById('messageInput').addEventListener('keypress', (e) => {
      if (e.key === 'Enter') this.sendMessage();
    });
    document.getElementById('welcomeUsernameInput').addEventListener('keypress', (e) => {
      if (e.key === 'Enter') {
        this.setWelcomeUsername();
      }
    });
  
  }

  switchMode(mode) {
    this.currentMode = mode;
    const chatHeader = document.getElementById('chatHeader');
    const usersList = document.getElementById('usersList');

    switch (mode) {
      case 'public':
        usersList.style.display = 'none';
        chatHeader.textContent = '公共聊天室';
        this.selectedUser = null;
        break;
      case 'private':
        usersList.style.display = 'block';
        this.api.requestUserList();
        this.addSystemMessage('正在获取在线用户列表...');
        break;
      case 'rename':
        document.getElementById('renameModal').style.display = 'flex';
        break;
    }
  }
}

// ---- 辅助函数 ----
function closeRenameModal() {
  document.getElementById('renameModal').style.display = 'none';
}

function updateUsername() {
  const newUsername = document.getElementById('newUsername').value.trim();
  if (newUsername) {
    chatClient.api.rename(newUsername);
    closeRenameModal();
    document.getElementById('newUsername').value = '';
  }
}
function setWelcomeUsername() {
  chatClient.setWelcomeUsername();
}

const chatClient = new ChatClient();
