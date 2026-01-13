// ========================================
// UI FUNCTIONS
// ========================================

function showMainMenu() {
    document.getElementById('mainMenu').style.display = 'block';
    document.getElementById('loginContainer').style.display = 'none';
    document.getElementById('registerContainer').style.display = 'none';
    document.getElementById('appContainer').style.display = 'none';
}

function showLogin() {
    document.getElementById('mainMenu').style.display = 'none';
    document.getElementById('loginContainer').style.display = 'block';
    document.getElementById('registerContainer').style.display = 'none';
    document.getElementById('appContainer').style.display = 'none';
    document.getElementById('uuidInput').value = '';
    document.getElementById('loginError').textContent = '';
    document.getElementById('loginSuccess').textContent = '';
    setTimeout(() => {
        document.getElementById('uuidInput').focus();
    }, 300);
}

function showRegisterForm() {
    document.getElementById('mainMenu').style.display = 'none';
    document.getElementById('loginContainer').style.display = 'none';
    document.getElementById('registerContainer').style.display = 'block';
    document.getElementById('appContainer').style.display = 'none';
    document.getElementById('registerError').textContent = '';
    document.getElementById('registerSuccess').textContent = '';
    
    // Reset UUID display state
    const uuidElement = document.getElementById('generatedUUID');
    const continueBtn = document.getElementById('continueBtn');
    const uuidBox = document.querySelector('.uuid-box');
    
    uuidElement.textContent = '';
    uuidElement.classList.remove('generating', 'masking', 'revealed');
    continueBtn.disabled = true;
    if (uuidBox) uuidBox.classList.add('show');
    
    // Start fake UUID animation immediately
    startFakeUUIDGeneration(
        uuidElement,
        document.getElementById('registerSuccess'),
        document.getElementById('uuidDisplay')
    );
}

function showRegister() {
    showRegisterForm();
}

function showApp() {
    document.getElementById('mainMenu').style.display = 'none';
    document.getElementById('loginContainer').style.display = 'none';
    document.getElementById('registerContainer').style.display = 'none';
    document.getElementById('appContainer').style.display = 'block';
    setTimeout(() => {
        document.getElementById('todoInput').focus();
    }, 300);
}

function renderTodos() {
    const todoList = document.getElementById('todoList');
    // Clear safely using removeChild instead of innerHTML
    while (todoList.firstChild) {
        todoList.removeChild(todoList.firstChild);
    }

    const filteredTodos = getFilteredAndSortedTodos();

    if (filteredTodos.length === 0) {
        const emptyMsg = document.createElement('li');
        emptyMsg.className = 'empty-message';
        emptyMsg.textContent = 'no todos found';
        todoList.appendChild(emptyMsg);
        return;
    }

    filteredTodos.forEach(todo => {
        const todoItem = document.createElement('li');
        // Validate priority before using in className to prevent XSS
        const priorityValidation = validatePriority(todo.priority || 'medium');
        const safePriority = priorityValidation.valid ? priorityValidation.sanitized : 'medium';
        todoItem.className = `todo-item priority-${safePriority}`;
        
        // Validate todo ID before setting dataset
        if (validateTodoId(todo.id)) {
            todoItem.dataset.todoId = String(todo.id);
            todoItem.dataset.id = String(todo.id);
        } else {
            console.error('Invalid todo ID in render:', todo.id);
            return; // Skip invalid todos
        }
        
        // Create checkbox
        const checkbox = document.createElement('input');
        checkbox.type = 'checkbox';
        checkbox.className = 'todo-checkbox';
        checkbox.checked = todo.completed;
        checkbox.addEventListener('change', () => toggleTodo(todo.id));
        
        // Create title span
        const titleSpan = document.createElement('span');
        titleSpan.className = `todo-title ${todo.completed ? 'completed' : ''}`;
        titleSpan.textContent = todo.title;
        titleSpan.addEventListener('dblclick', () => startInlineEdit(titleSpan));
        
        // Create edit input
        const editInput = document.createElement('input');
        editInput.type = 'text';
        editInput.className = 'todo-edit-input';
        editInput.style.display = 'none';
        editInput.addEventListener('blur', () => finishInlineEdit(editInput));
        editInput.addEventListener('keydown', (e) => handleEditKeydown(e, editInput));
        
        // Create meta group
        const metaGroup = document.createElement('span');
        metaGroup.className = 'todo-meta-group';
        
        // Priority indicator - only show for high
        if (todo.priority === 'high') {
            const prioritySpan = document.createElement('span');
            prioritySpan.className = 'todo-meta todo-priority-high';
            prioritySpan.textContent = '!';
            metaGroup.appendChild(prioritySpan);
        }
        
        // Due date
        if (todo.due_date) {
            const dueDate = new Date(todo.due_date);
            const today = new Date();
            today.setHours(0, 0, 0, 0);
            const isOverdue = dueDate < today && !todo.completed;
            const isToday = dueDate.toDateString() === today.toDateString();
            const dateStr = dueDate.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
            
            const dueDateSpan = document.createElement('span');
            dueDateSpan.className = `todo-meta todo-due-date ${isOverdue ? 'overdue' : (isToday ? 'today' : '')}`;
            dueDateSpan.textContent = dateStr;
            metaGroup.appendChild(dueDateSpan);
        }
        
        // Tags
        if (todo.tags && todo.tags.length > 0) {
            const tagsContainer = document.createElement('span');
            tagsContainer.className = 'todo-tags-inline';
            todo.tags.forEach(tag => {
                const tagSpan = document.createElement('span');
                tagSpan.className = 'todo-tag';
                tagSpan.textContent = '#' + tag;
                tagSpan.addEventListener('click', () => filterByTag(tag));
                tagsContainer.appendChild(tagSpan);
            });
            metaGroup.appendChild(tagsContainer);
        }
        
        // Create actions div
        const actionsDiv = document.createElement('div');
        actionsDiv.className = 'todo-actions';
        
        const editBtn = document.createElement('button');
        editBtn.className = 'action-btn edit-btn';
        editBtn.title = 'Edit';
        editBtn.textContent = '✎';
        editBtn.addEventListener('click', () => startInlineEditById(todo.id));
        
        const moreBtn = document.createElement('button');
        moreBtn.className = 'action-btn more-btn';
        moreBtn.title = 'More options';
        moreBtn.textContent = '⋯';
        moreBtn.addEventListener('click', (e) => showTodoMenu(e, todo.id));
        
        const deleteBtn = document.createElement('button');
        deleteBtn.className = 'action-btn delete-btn';
        deleteBtn.title = 'Delete';
        deleteBtn.textContent = '×';
        deleteBtn.addEventListener('click', () => deleteTodo(todo.id));
        
        actionsDiv.appendChild(editBtn);
        actionsDiv.appendChild(moreBtn);
        actionsDiv.appendChild(deleteBtn);
        
        // Assemble todo item
        todoItem.appendChild(checkbox);
        todoItem.appendChild(titleSpan);
        todoItem.appendChild(editInput);
        todoItem.appendChild(metaGroup);
        todoItem.appendChild(actionsDiv);
        
        todoList.appendChild(todoItem);
    });
}

function getFilteredTodos() {
    // This is kept for backward compatibility but getFilteredAndSortedTodos is preferred
    return getFilteredAndSortedTodos();
}

function filterTodos(filter) {
    const filterValidation = validateFilter(filter);
    if (!filterValidation.valid) {
        showError(filterValidation.error);
        return;
    }
    
    currentFilter = filterValidation.sanitized;
    
    // Update button states
    document.querySelectorAll('.filter-btn').forEach(btn => {
        btn.classList.remove('active');
    });
    const filterBtn = document.getElementById(`filter${currentFilter.charAt(0).toUpperCase() + currentFilter.slice(1)}`);
    if (filterBtn) {
        filterBtn.classList.add('active');
    }
    
    renderTodos();
}

function updateTodoCount() {
    const activeCount = todos.filter(todo => !todo.completed).length;
    const text = `${activeCount} item${activeCount !== 1 ? 's' : ''} left`;
    document.getElementById('todoCount').textContent = text;
}