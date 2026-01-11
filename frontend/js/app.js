// ========================================
// MAIN APPLICATION INITIALIZATION
// ========================================

// Initialize event listeners when DOM is ready
document.addEventListener('DOMContentLoaded', function() {
    // Check if user is already logged in
    const savedToken = localStorage.getItem('jwtToken');
    const savedUUID = localStorage.getItem('userUUID');
    
    if (savedToken) {
        userUUID = savedUUID;
        showApp();
        loadTodos();
    } else {
        showMainMenu();
    }

    setupEventListeners();
});

function setupEventListeners() {
    // Main menu event listeners
    document.getElementById('showLoginBtn').addEventListener('click', showLogin);
    document.getElementById('showRegisterFormBtn').addEventListener('click', showRegisterForm);
    document.getElementById('backToMenuFromLogin').addEventListener('click', showMainMenu);

    // Brand title click - go back to main menu
    const loginBrandTitle = document.getElementById('loginBrandTitle');
    const registerBrandTitle = document.getElementById('registerBrandTitle');
    if (loginBrandTitle) {
        loginBrandTitle.addEventListener('click', showMainMenu);
    }
    if (registerBrandTitle) {
        registerBrandTitle.addEventListener('click', function() {
            // Cancel any pending registration first
            cancelRegistration();
        });
    }

    // Login page event listeners
    document.getElementById('loginBtn').addEventListener('click', login);
    document.getElementById('uuidInput').addEventListener('keypress', function(e) {
        if (e.key === 'Enter') login();
    });
    
    // Eye icon hover to reveal/hide UUID (handled by eye-hover-animation.js)
    // const eyeIcon = document.getElementById('eyeIcon');
    // const uuidInput = document.getElementById('uuidInput');
    // if (eyeIcon && uuidInput) {
    //     eyeIcon.addEventListener('mouseenter', function() {
    //         uuidInput.type = 'text';
    //     });
    //     eyeIcon.addEventListener('mouseleave', function() {
    //         uuidInput.type = 'password';
    //     });
    // }

    // Register page event listeners
    document.getElementById('copyBtn').addEventListener('click', copyUUID);
    document.getElementById('continueBtn').addEventListener('click', continueWithUUID);
    document.getElementById('cancelRegistration').addEventListener('click', cancelRegistration);

    // App page event listeners
    document.getElementById('logoutBtn').addEventListener('click', logout);
    document.getElementById('addTodoBtn').addEventListener('click', addTodo);
    
    // Textarea: Enter to add todo
    const todoInput = document.getElementById('todoInput');
    todoInput.addEventListener('keydown', function(e) {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            addTodo();
        }
    });

    // Filter buttons
    document.getElementById('filterAll').addEventListener('click', function() { filterTodos('all'); });
    document.getElementById('filterActive').addEventListener('click', function() { filterTodos('active'); });
    document.getElementById('filterCompleted').addEventListener('click', function() { filterTodos('completed'); });

    // Sort select
    const sortSelect = document.getElementById('sortSelect');
    if (sortSelect) {
        sortSelect.addEventListener('change', function() {
            setSortOrder(this.value);
        });
    }

    // Clear completed
    document.getElementById('clearCompletedBtn').addEventListener('click', clearCompleted);
    
    // Close todo menu when clicking outside
    document.addEventListener('click', function(e) {
        const menu = document.querySelector('.todo-context-menu');
        if (menu && !e.target.closest('.todo-context-menu') && !e.target.closest('.more-btn')) {
            menu.remove();
        }
    });
}

// ========================================
// TODO CONTEXT MENU
// ========================================

function showTodoMenu(event, todoId) {
    event.stopPropagation();
    
    // Remove any existing menu
    const existingMenu = document.querySelector('.todo-context-menu');
    if (existingMenu) existingMenu.remove();
    
    const todo = todos.find(t => t.id === todoId);
    if (!todo) return;
    
    const menu = document.createElement('div');
    menu.className = 'todo-context-menu';
    
    // Priority section
    const prioritySection = document.createElement('div');
    prioritySection.className = 'menu-section';
    
    const priorityLabel = document.createElement('label');
    priorityLabel.className = 'menu-label';
    priorityLabel.textContent = 'priority';
    prioritySection.appendChild(priorityLabel);
    
    const priorityOptions = document.createElement('div');
    priorityOptions.className = 'menu-options';
    
    ['low', 'medium', 'high'].forEach(p => {
        const btn = document.createElement('button');
        btn.className = `menu-opt ${todo.priority === p ? 'active' : ''}`;
        btn.textContent = p === 'medium' ? 'med' : p;
        btn.addEventListener('click', () => setTodoPriority(todoId, p));
        priorityOptions.appendChild(btn);
    });
    prioritySection.appendChild(priorityOptions);
    menu.appendChild(prioritySection);
    
    // Due date section
    const dateSection = document.createElement('div');
    dateSection.className = 'menu-section';
    
    const dateLabel = document.createElement('label');
    dateLabel.className = 'menu-label';
    dateLabel.textContent = 'due date';
    dateSection.appendChild(dateLabel);
    
    const dateInput = document.createElement('input');
    dateInput.type = 'date';
    dateInput.className = 'menu-date';
    dateInput.value = todo.due_date ? todo.due_date.split('T')[0] : '';
    dateInput.addEventListener('change', () => setTodoDueDate(todoId, dateInput.value));
    dateSection.appendChild(dateInput);
    menu.appendChild(dateSection);
    
    // Tags section
    const tagsSection = document.createElement('div');
    tagsSection.className = 'menu-section';
    
    const tagsLabel = document.createElement('label');
    tagsLabel.className = 'menu-label';
    tagsLabel.textContent = 'tags';
    tagsSection.appendChild(tagsLabel);
    
    const tagsInput = document.createElement('input');
    tagsInput.type = 'text';
    tagsInput.className = 'menu-tags';
    tagsInput.placeholder = 'comma separated';
    tagsInput.value = (todo.tags || []).join(', ');
    tagsInput.addEventListener('change', () => setTodoTags(todoId, tagsInput.value));
    tagsSection.appendChild(tagsInput);
    menu.appendChild(tagsSection);
    
    // Position menu near the button
    const btn = event.target;
    const rect = btn.getBoundingClientRect();
    menu.style.position = 'fixed';
    menu.style.top = (rect.bottom + 5) + 'px';
    menu.style.right = (window.innerWidth - rect.right) + 'px';
    
    document.body.appendChild(menu);
}

async function setTodoPriority(todoId, priority) {
    const todo = todos.find(t => t.id === todoId);
    if (!todo) return;
    await updateTodo(todoId, { priority });
}

async function setTodoDueDate(todoId, dateStr) {
    const todo = todos.find(t => t.id === todoId);
    if (!todo) return;
    const due_date = dateStr ? dateStr : null;
    await updateTodo(todoId, { due_date });
}

async function setTodoTags(todoId, tagsStr) {
    const todo = todos.find(t => t.id === todoId);
    if (!todo) return;
    const tags = tagsStr ? tagsStr.split(',').map(t => t.trim()).filter(t => t) : [];
    await updateTodo(todoId, { tags });
}