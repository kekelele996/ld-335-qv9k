-- ld-335 GbInsureAPI 医保智能结算API网关 初始化脚本（容器首次启动自动执行）
CREATE TABLE IF NOT EXISTS api_clients (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    client_type VARCHAR(20) NOT NULL,
    api_key_hash VARCHAR(128) NOT NULL,
    role VARCHAR(50) NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    rate_limit_qps INT DEFAULT 10,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS insured_persons (
    id BIGSERIAL PRIMARY KEY,
    id_card_no VARCHAR(18) UNIQUE NOT NULL,
    medical_card_no VARCHAR(32) UNIQUE NOT NULL,
    name VARCHAR(50) NOT NULL,
    insurance_type VARCHAR(20) NOT NULL,
    insurance_status VARCHAR(20) NOT NULL,
    insurance_place VARCHAR(100) DEFAULT '',
    personal_balance DOUBLE PRECISION DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS upload_batches (
    id BIGSERIAL PRIMARY KEY,
    batch_no VARCHAR(32) UNIQUE NOT NULL,
    client_id BIGINT NOT NULL,
    insured_person_id BIGINT NOT NULL,
    total_amount DOUBLE PRECISION DEFAULT 0,
    item_count INT DEFAULT 0,
    upload_status VARCHAR(20) DEFAULT 'validating',
    created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_batch_client ON upload_batches(client_id);

CREATE TABLE IF NOT EXISTS fee_items (
    id BIGSERIAL PRIMARY KEY,
    batch_id BIGINT NOT NULL,
    item_code VARCHAR(32) NOT NULL,
    item_name VARCHAR(100) NOT NULL,
    item_type VARCHAR(20) NOT NULL,
    unit_price DOUBLE PRECISION DEFAULT 0,
    quantity DOUBLE PRECISION DEFAULT 0,
    amount DOUBLE PRECISION DEFAULT 0,
    self_pay_ratio DOUBLE PRECISION DEFAULT 0,
    medical_category VARCHAR(20) NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_fee_batch ON fee_items(batch_id);

CREATE TABLE IF NOT EXISTS presettlements (
    id BIGSERIAL PRIMARY KEY,
    batch_id BIGINT NOT NULL,
    insured_person_id BIGINT NOT NULL,
    total_amount DOUBLE PRECISION DEFAULT 0,
    insurance_pay_amount DOUBLE PRECISION DEFAULT 0,
    personal_account_amount DOUBLE PRECISION DEFAULT 0,
    self_pay_amount DOUBLE PRECISION DEFAULT 0,
    deductible DOUBLE PRECISION DEFAULT 0,
    reimbursement_ratio DOUBLE PRECISION DEFAULT 0,
    result_payload TEXT DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_preset_batch ON presettlements(batch_id);

CREATE TABLE IF NOT EXISTS settlement_orders (
    id BIGSERIAL PRIMARY KEY,
    settlement_no VARCHAR(32) UNIQUE NOT NULL,
    batch_id BIGINT NOT NULL,
    insured_person_id BIGINT NOT NULL,
    presettlement_id BIGINT NOT NULL,
    client_id BIGINT NOT NULL,
    status VARCHAR(20) DEFAULT 'presettled',
    total_amount DOUBLE PRECISION DEFAULT 0,
    insurance_pay_amount DOUBLE PRECISION DEFAULT 0,
    settled_at TIMESTAMPTZ,
    reversed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_order_client ON settlement_orders(client_id, status);

CREATE TABLE IF NOT EXISTS daily_reconciliations (
    id BIGSERIAL PRIMARY KEY,
    reconcile_date VARCHAR(10) UNIQUE NOT NULL,
    total_count BIGINT DEFAULT 0,
    total_amount DOUBLE PRECISION DEFAULT 0,
    success_count BIGINT DEFAULT 0,
    fail_count BIGINT DEFAULT 0,
    abnormal_orders BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    client_id BIGINT DEFAULT 0,
    method VARCHAR(10) DEFAULT '',
    path VARCHAR(200) DEFAULT '',
    status_code INT DEFAULT 0,
    latency_ms BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_audit_client ON audit_logs(client_id);

-- 种子数据（api_key_hash 为明文 API Key 的 HMAC 哈希：ak_demo_his / ak_demo_third）
INSERT INTO api_clients (name, client_type, api_key_hash, role, status, rate_limit_qps) VALUES
('演示医院 HIS', 'HIS', 'demo-his-hash-placeholder', 'settlement', 'active', 20),
('第三方药房', 'THIRD_PARTY', 'demo-third-hash-placeholder', 'query', 'active', 5)
ON CONFLICT DO NOTHING;

INSERT INTO insured_persons (id_card_no, medical_card_no, name, insurance_type, insurance_status, insurance_place, personal_balance) VALUES
('110101199001011234', 'M110101199001011234', '张三', 'employee', 'active', '北京市', 3200),
('310101198505052345', 'M310101198505052345', '李四', 'resident', 'resident', '上海市', 1500),
('440101197803036789', 'M440101197803036789', '王五', 'new_rural', 'active', '广州市', 600)
ON CONFLICT (id_card_no) DO NOTHING;
