document.addEventListener('DOMContentLoaded', function () {
    const orderUidInput = document.getElementById('orderUid');
    const getOrderBtn = document.getElementById('getOrderBtn');
    const loadingDiv = document.getElementById('loading');
    const errorDiv = document.getElementById('error');
    const orderResultDiv = document.getElementById('orderResult');

    getOrderBtn.addEventListener('click', async function (e) {
        e.preventDefault()

        const orderUid = orderUidInput.value.trim();
        if (!orderUid) {
            showError('Пожалуйста, введите ID заказа');
            return;
        }

        hideAll();
        showLoading();

        try {
            const response = await fetch(`/order/${encodeURIComponent(orderUid)}`);

            if (!response.ok) {
                const errorText = await response.text();
                showError(`${response.status} - ${errorText}`);
                return;
            }
            const order = await response.json();
            displayOrder(order);
        } catch (error) {
            console.error('Error fetching order:', error);
            showError(`Ошибка: ${error.message}`);
        } finally {
            hideLoading();
        }
    });

    function showLoading() {
        loadingDiv.classList.remove('hidden');
        errorDiv.classList.add('hidden');
        orderResultDiv.classList.add('hidden');
    }

    function hideLoading() {
        loadingDiv.classList.add('hidden');
    }

    function showError(message) {
        errorDiv.textContent = message;
        errorDiv.classList.remove('hidden');
        orderResultDiv.classList.add('hidden');
        loadingDiv.classList.add('hidden');
    }

    function hideAll() {
        loadingDiv.classList.add('hidden');
        errorDiv.classList.add('hidden');
        orderResultDiv.classList.add('hidden');
    }

    function displayOrder(order) {
        orderResultDiv.innerHTML = `
            <div class="order-header">
                <h3>📦 Заказ: </h3>
                <p><strong>Статус:</strong> ${getStatusText(order.items[0]?.status || 'unknown')}</p>
            </div>
            
            <div class="order-details">
                <div class="detail-card">
                    <h4>🚚 Доставка</h4>
                    <div class="detail-item">
                        <span class="detail-label">Имя:</span>
                        <span class="detail-value">${order.delivery?.name || '-'}</span>
                    </div>
                    <div class="detail-item">
                        <span class="detail-label">Телефон:</span>
                        <span class="detail-value">${order.delivery?.phone || '-'}</span>
                    </div>
                    <div class="detail-item">
                        <span class="detail-label">Адрес:</span>
                        <span class="detail-value">${order.delivery?.address || '-'}</span>
                    </div>
                    <div class="detail-item">
                        <span class="detail-label">Город:</span>
                        <span class="detail-value">${order.delivery?.city || '-'}</span>
                    </div>
                    <div class="detail-item">
                        <span class="detail-label">Индекс:</span>
                        <span class="detail-value">${order.delivery?.zip || '-'}</span>
                    </div>
                    <div class="detail-item">
                        <span class="detail-label">Регион:</span>
                        <span class="detail-value">${order.delivery?.region || '-'}</span>
                    </div>
                    <div class="detail-item">
                        <span class="detail-label">Email:</span>
                        <span class="detail-value">${order.delivery?.email || '-'}</span>
                    </div>
                </div>
                
                <div class="detail-card">
                    <h4>💳 Оплата</h4>
                    <div class="detail-item">
                        <span class="detail-label">Сумма:</span>
                        <span class="detail-value">${formatCurrency(order.payment?.amount || 0)}</span>
                    </div>
                    <div class="detail-item">
                        <span class="detail-label">Валюта:</span>
                        <span class="detail-value">${order.payment?.currency || '-'}</span>
                    </div>
                    <div class="detail-item">
                        <span class="detail-label">Провайдер:</span>
                        <span class="detail-value">${order.payment?.provider || '-'}</span>
                    </div>
                    <div class="detail-item">
                        <span class="detail-label">Транзакция:</span>
                        <span class="detail-value">${order.payment?.transaction || '-'}</span>
                    </div>
                    <div class="detail-item">
                        <span class="detail-label">Стоимость доставки:</span>
                        <span class="detail-value">${formatCurrency(order.payment?.delivery_cost || 0)}</span>
                    </div>
                    <div class="detail-item">
                        <span class="detail-label">Сумма товаров:</span>
                        <span class="detail-value">${formatCurrency(order.payment?.goods_total || 0)}</span>
                    </div>
                    <div class="detail-item">
                        <span class="detail-label">Комиссия:</span>
                        <span class="detail-value">${formatCurrency(order.payment?.custom_fee || 0)}</span>
                    </div>
                </div>
                
                <div class="detail-card">
                    <h4>📋 Общая информация</h4>
                    <div class="detail-item">
                        <span class="detail-label">Track Number:</span>
                        <span class="detail-value">${order.track_number || '-'}</span>
                    </div>
                    <div class="detail-item">
                        <span class="detail-label">Entry:</span>
                        <span class="detail-value">${order.entry || '-'}</span>
                    </div>
                    <div class="detail-item">
                        <span class="detail-label">Locale:</span>
                        <span class="detail-value">${order.locale || '-'}</span>
                    </div>
                    <div class="detail-item">
                        <span class="detail-label">Дата создания:</span>
                        <span class="detail-value">${formatDate(order.date_created) || '-'}</span>
                    </div>
                    <div class="detail-item">
                        <span class="detail-label">Служба доставки:</span>
                        <span class="detail-value">${order.delivery_service || '-'}</span>
                    </div>
                </div>
            </div>
            
            <div class="items-list">
                <h4>🛍️ Товары (${order.items.length})</h4>
                ${order.items.map(item => `
                    <div class="item-card">
                        <div class="item-grid">
                            <div class="detail-item">
                                <span class="detail-label">Название:</span>
                                <span class="detail-value">${item.name || '-'}</span>
                            </div>
                            <div class="detail-item">
                                <span class="detail-label">Бренд:</span>
                                <span class="detail-value">${item.brand || '-'}</span>
                            </div>
                            <div class="detail-item">
                                <span class="detail-label">Размер:</span>
                                <span class="detail-value">${item.size || '-'}</span>
                            </div>
                            <div class="detail-item">
                                <span class="detail-label">Цена:</span>
                                <span class="detail-value">${formatCurrency(item.price)}</span>
                            </div>
                            <div class="detail-item">
                                <span class="detail-label">Скидка:</span>
                                <span class="detail-value">${item.sale}%</span>
                            </div>
                            <div class="detail-item">
                                <span class="detail-label">Итого:</span>
                                <span class="detail-value">${formatCurrency(item.total_price)}</span>
                            </div>
                            <div class="detail-item">
                                <span class="detail-label">Статус:</span>
                                <span class="detail-value">${getStatusText(item.status)}</span>
                            </div>
                        </div>
                    </div>
                `).join('')}
            </div>
        `;

        orderResultDiv.classList.remove('hidden');
    }

    function formatCurrency(amount) {
        if (!amount) return '0';
        return `${amount} руб.`;
    }

    function formatDate(dateString) {
        if (!dateString) return '-';
        const date = new Date(dateString);
        return date.toLocaleDateString('ru-RU') + ' ' + date.toLocaleTimeString('ru-RU');
    }

    function getStatusText(status) {
        const statusMap = {
            'pending': 'Ожидание',
            'processing': 'Обработка',
            'assembling': 'Сборка',
            'in_transit': 'В пути',
            'delivered': 'Доставлен',
            'cancelled': 'Отменен',
            'returned': 'Возвращен',
            '202': 'Доставлен'  // WB статус
        };
        return statusMap[status] || status;
    }
});