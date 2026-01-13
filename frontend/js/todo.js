// ========================================
// TODO FUNCTIONS
// ========================================

// Sort state
let currentSort = 'created';
let currentTagFilter = null;

async function loadTodos() {
    try {
        const response = await fetch(`${API_BASE_URL}/todos`, {
            headers: getAuthHeaders()
        });

        if (response.ok) {
            const data = await response.json();
            
            if (Array.isArray(data)) {
                todos = data;
                renderTagFilters();
                renderTodos();
                updateTodoCount();
            } else {
                console.error('Invalid todos data');
            }
        } else if (response.status === 401) {
            sessionStorage.removeItem('jwtToken');
            logout();
            showError('Session expired. Please login again.');
        } else {
            console.error('Failed to load todos');
            showError('Failed to load todos. Please try again.');
        }
    } catch (error) {
        console.error('Error loading todos:', error);
        showError('Network error. Unable to load todos.');
    }
}

async function addTodo() {
    const input = document.getElementById('todoInput');
    const prioritySelect = document.getElementById('prioritySelect');
    const dueDateInput = document.getElementById('dueDateInput');
    const tagsInput = document.getElementById('tagsInput');
    
    const rawTitle = input.value;

    // Frontend validation
    const validation = validateTodoInput(rawTitle);
    if (!validation.valid) {
        showError(validation.error);
        return;
    }

    const title = validation.sanitized;
    
    // Validate priority against whitelist
    const priorityValidation = validatePriority(prioritySelect.value || 'medium');
    if (!priorityValidation.valid) {
        showError(priorityValidation.error);
        return;
    }
    const priority = priorityValidation.sanitized;
    
    // Validate date
    const dueDateValidation = validateDate(dueDateInput.value || null);
    if (!dueDateValidation.valid) {
        showError(dueDateValidation.error);
        return;
    }
    const dueDate = dueDateValidation.sanitized;
    
    // Validate tags
    const rawTags = tagsInput.value ? tagsInput.value.split(',').map(t => t.trim()).filter(t => t) : [];
    const tagsValidation = validateTags(rawTags);
    if (!tagsValidation.valid) {
        showError(tagsValidation.error);
        return;
    }
    const tags = tagsValidation.sanitized;

    try {
        const body = { 
            title, 
            completed: false,
            priority,
            tags
        };
        if (dueDate) {
            body.due_date = dueDate;
        }

        const response = await fetch(`${API_BASE_URL}/todos`, {
            method: 'POST',
            headers: getAuthHeaders(),
            body: JSON.stringify(body)
        });

        if (response.ok) {
            const data = await response.json();
            
            if (data) {
                todos.push(data);
                input.value = '';
                tagsInput.value = '';
                dueDateInput.value = '';
                prioritySelect.value = 'medium';
                renderTagFilters();
                renderTodos();
                updateTodoCount();
                showSuccess('Todo successfully added');
            }
        } else if (response.status === 401) {
            sessionStorage.removeItem('jwtToken');
            logout();
            showError('Session expired. Please login again.');
        } else {
            console.error('Failed to create todo');
            showError('Failed to add todo. Please try again.');
        }
    } catch (error) {
        console.error('Error creating todo:', error);
        showError('Network error. Please check your connection and try again.');
    }
}

async function toggleTodo(id) {
    if (!validateTodoId(id)) {
        showError('Invalid todo ID');
        return;
    }
    
    const todo = todos.find(t => t.id === id);
    if (!todo) return;

    try {
        const response = await fetch(`${API_BASE_URL}/todos/${id}`, {
            method: 'PUT',
            headers: getAuthHeaders(),
            body: JSON.stringify({
                title: todo.title,
                completed: !todo.completed
            })
        });

        if (response.ok) {
            const data = await response.json();
            
            if (data) {
                const index = todos.findIndex(t => t.id === id);
                todos[index] = data;
                renderTodos();
                updateTodoCount();
                if (data.completed) {
                    showSuccess('Todo marked as completed');
                }
            }
        } else if (response.status === 401) {
            sessionStorage.removeItem('jwtToken');
            logout();
            showError('Session expired. Please login again.');
        } else {
            console.error('Failed to update todo');
            showError('Failed to update todo. Please try again.');
        }
    } catch (error) {
        console.error('Error updating todo:', error);
        showError('Network error. Please check your connection and try again.');
    }
}

async function deleteTodo(id) {
    if (!validateTodoId(id)) {
        showError('Invalid todo ID');
        return;
    }
    
    try {
        const response = await fetch(`${API_BASE_URL}/todos/${id}`, {
            method: 'DELETE',
            headers: getAuthHeaders()
        });

        if (response.ok || response.status === 204) {
            todos = todos.filter(t => t.id !== id);
            renderTodos();
            updateTodoCount();
            showSuccess('Todo successfully deleted');
        } else if (response.status === 401) {
            sessionStorage.removeItem('jwtToken');
            logout();
            showError('Session expired. Please login again.');
        } else {
            console.error('Failed to delete todo');
            showError('Failed to delete todo. Please try again.');
        }
    } catch (error) {
        console.error('Error deleting todo:', error);
        showError('Network error. Please check your connection and try again.');
    }
}

// Modern inline editing instead of prompt()
function startInlineEdit(titleElement) {
    if (titleElement.style.display === 'none') return; // Already editing
    
    const todoItem = titleElement.closest('.todo-item');
    if (!todoItem) return;
    
    const todoId = todoItem.dataset.todoId;
    if (!validateTodoId(todoId)) {
        showError('Invalid todo ID');
        return;
    }
    
    const todo = todos.find(t => t.id === todoId);
    if (!todo) return;
    
    const editInput = titleElement.nextElementSibling;
    
    // Set up edit input
    editInput.value = todo.title;
    editInput.style.display = 'block';
    titleElement.style.display = 'none';
    
    // Focus and select text
    editInput.focus();
    editInput.select();
}

// Start inline edit by todo ID (for edit button)
function startInlineEditById(todoId) {
    if (!validateTodoId(todoId)) {
        showError('Invalid todo ID');
        return;
    }
    
    // Use escapeHtml to prevent XSS in querySelector (defense-in-depth)
    // Since todoId is validated, this is extra safety
    const escapedId = escapeHtml(String(todoId));
    const todoItem = document.querySelector(`.todo-item[data-todo-id="${escapedId}"]`);
    if (!todoItem) return;
    
    const titleElement = todoItem.querySelector('.todo-title');
    if (titleElement) {
        startInlineEdit(titleElement);
    }
}

function finishInlineEdit(editInput) {
    const titleElement = editInput.previousElementSibling;
    const todoItem = editInput.closest('.todo-item');
    if (!todoItem) return;
    
    const todoId = todoItem.dataset.todoId;
    if (!validateTodoId(todoId)) {
        showError('Invalid todo ID');
        return;
    }
    
    const newTitle = editInput.value.trim();
    
    // Validation
    const validation = validateTodoInput(newTitle);
    if (!validation.valid) {
        showError(validation.error);
        editInput.focus();
        return;
    }
    
    // Update todo
    updateTodoTitle(todoId, validation.sanitized);
    
    // Hide edit input and show title
    editInput.style.display = 'none';
    titleElement.style.display = 'inline';
}

function handleEditKeydown(event, editInput) {
    if (event.key === 'Enter') {
        event.preventDefault();
        editInput.blur(); // This will trigger finishInlineEdit
    } else if (event.key === 'Escape') {
        // Cancel editing
        const titleElement = editInput.previousElementSibling;
        editInput.style.display = 'none';
        titleElement.style.display = 'inline';
    }
}

async function updateTodoTitle(id, newTitle) {
    if (!validateTodoId(id)) {
        showError('Invalid todo ID');
        return;
    }
    
    const todo = todos.find(t => t.id === id);
    if (!todo) return;

    try {
        const response = await fetch(`${API_BASE_URL}/todos/${id}`, {
            method: 'PUT',
            headers: getAuthHeaders(),
            body: JSON.stringify({
                title: newTitle,
                completed: todo.completed
            })
        });

        if (response.ok) {
            const data = await response.json();
            
            if (data) {
                const index = todos.findIndex(t => t.id === id);
                todos[index] = data;
                renderTodos();
                showSuccess('Todo successfully updated');
            }
        } else if (response.status === 401) {
            sessionStorage.removeItem('jwtToken');
            logout();
            showError('Session expired. Please login again.');
        } else {
            console.error('Failed to update todo');
            showError('Failed to update todo. Please try again.');
        }
    } catch (error) {
        console.error('Error updating todo:', error);
        showError('Network error. Please check your connection and try again.');
    }
}

// Generic update function for context menu
async function updateTodo(id, updates) {
    if (!validateTodoId(id)) {
        showError('Invalid todo ID');
        return;
    }
    
    const todo = todos.find(t => t.id === id);
    if (!todo) return;

    try {
        // Validate priority if provided
        let priority = todo.priority;
        if (updates.priority !== undefined) {
            const priorityValidation = validatePriority(updates.priority);
            if (!priorityValidation.valid) {
                showError(priorityValidation.error);
                return;
            }
            priority = priorityValidation.sanitized;
        }
        
        // Validate tags if provided
        let tags = todo.tags || [];
        if (updates.tags !== undefined) {
            const tagsValidation = validateTags(updates.tags);
            if (!tagsValidation.valid) {
                showError(tagsValidation.error);
                return;
            }
            tags = tagsValidation.sanitized;
        }
        
        // Validate due_date if provided
        let due_date = null;
        if (updates.due_date !== undefined) {
            const dateValidation = validateDate(updates.due_date);
            if (!dateValidation.valid) {
                showError(dateValidation.error);
                return;
            }
            due_date = dateValidation.sanitized;
        } else if (todo.due_date) {
            // Extract date part from ISO datetime string
            due_date = todo.due_date.split('T')[0];
        }
        
        const body = {
            title: todo.title,
            completed: todo.completed,
            priority,
            tags,
            due_date
        };

        const response = await fetch(`${API_BASE_URL}/todos/${id}`, {
            method: 'PUT',
            headers: getAuthHeaders(),
            body: JSON.stringify(body)
        });

        if (response.ok) {
            const data = await response.json();
            
            if (data) {
                const index = todos.findIndex(t => t.id === id);
                todos[index] = data;
                renderTagFilters();
                renderTodos();
                
                // Close context menu
                const menu = document.querySelector('.todo-context-menu');
                if (menu) menu.remove();
                
                // Show success notification
                showSuccess('Todo successfully updated');
            }
        } else if (response.status === 401) {
            sessionStorage.removeItem('jwtToken');
            logout();
            showError('Session expired. Please login again.');
        } else {
            console.error('Failed to update todo');
            showError('Failed to update todo. Please try again.');
        }
    } catch (error) {
        console.error('Error updating todo:', error);
        showError('Network error. Please check your connection and try again.');
    }
}

async function clearCompleted() {
    const completedTodos = todos.filter(t => t.completed);
    
    if (completedTodos.length === 0) {
        showWarning('No completed todos to clear');
        return;
    }
    
    for (const todo of completedTodos) {
        await deleteTodo(todo.id);
    }
    
    showSuccess(`${completedTodos.length} completed todo${completedTodos.length > 1 ? 's' : ''} cleared`);
}

// ========================================
// SORT AND TAG FILTER FUNCTIONS
// ========================================

function setSortOrder(sort) {
    const sortValidation = validateSortOrder(sort);
    if (!sortValidation.valid) {
        showError(sortValidation.error);
        return;
    }
    currentSort = sortValidation.sanitized;
    renderTodos();
}

function filterByTag(tag) {
    // Validate tag if provided (null is allowed to clear filter)
    if (tag !== null && tag !== undefined) {
        const tagValidation = validateTag(tag);
        if (!tagValidation.valid) {
            showError('Invalid tag');
            return;
        }
        tag = tagValidation.sanitized;
    }
    
    if (currentTagFilter === tag) {
        currentTagFilter = null; // Toggle off
    } else {
        currentTagFilter = tag;
    }
    renderTagFilters();
    renderTodos();
}

function getAllTags() {
    const tagSet = new Set();
    todos.forEach(todo => {
        if (todo.tags && Array.isArray(todo.tags)) {
            todo.tags.forEach(tag => tagSet.add(tag));
        }
    });
    return Array.from(tagSet).sort();
}

function renderTagFilters() {
    const container = document.getElementById('tagFilters');
    if (!container) return;

    const allTags = getAllTags();
    
    if (allTags.length === 0) {
        // Clear safely using removeChild instead of innerHTML
        while (container.firstChild) {
            container.removeChild(container.firstChild);
        }
        return;
    }

    // Clear and rebuild with event listeners
    while (container.firstChild) {
        container.removeChild(container.firstChild);
    }
    
    const label = document.createElement('span');
    label.className = 'tag-filter-label';
    label.textContent = 'tags:';
    container.appendChild(label);
    
    allTags.forEach(tag => {
        const isActive = currentTagFilter === tag;
        const btn = document.createElement('button');
        btn.className = `tag-filter-btn ${isActive ? 'active' : ''}`;
        btn.textContent = tag;
        btn.addEventListener('click', () => filterByTag(tag));
        container.appendChild(btn);
    });

    if (currentTagFilter) {
        const clearBtn = document.createElement('button');
        clearBtn.className = 'tag-filter-clear';
        clearBtn.textContent = '✕';
        clearBtn.addEventListener('click', () => filterByTag(null));
        container.appendChild(clearBtn);
    }
}

function getFilteredAndSortedTodos() {
    let result = [...todos];

    // Apply status filter (all/active/completed)
    switch (currentFilter) {
        case 'active':
            result = result.filter(todo => !todo.completed);
            break;
        case 'completed':
            result = result.filter(todo => todo.completed);
            break;
    }

    // Apply tag filter
    if (currentTagFilter) {
        result = result.filter(todo => 
            todo.tags && Array.isArray(todo.tags) && todo.tags.includes(currentTagFilter)
        );
    }

    // Apply sorting
    result.sort((a, b) => {
        switch (currentSort) {
            case 'due':
                // Todos without due date go to end
                if (!a.due_date && !b.due_date) return 0;
                if (!a.due_date) return 1;
                if (!b.due_date) return -1;
                return new Date(a.due_date) - new Date(b.due_date);
            case 'priority':
                const priorityOrder = { high: 0, medium: 1, low: 2 };
                return (priorityOrder[a.priority] || 1) - (priorityOrder[b.priority] || 1);
            case 'alpha':
                return a.title.localeCompare(b.title);
            case 'created':
            default:
                return new Date(b.created_at) - new Date(a.created_at);
        }
    });

    return result;
}