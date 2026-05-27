-- Shared updated_at trigger
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Turfs
CREATE TABLE IF NOT EXISTS turfs (
  id              BIGSERIAL PRIMARY KEY,
  name            TEXT NOT NULL,
  description     TEXT NOT NULL DEFAULT '',
  address         TEXT NOT NULL DEFAULT '',
  price_per_hour  NUMERIC NOT NULL DEFAULT 0,
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at      TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_turfs_updated_at ON turfs(updated_at);
DROP TRIGGER IF EXISTS trg_turfs_updated_at ON turfs;
CREATE TRIGGER trg_turfs_updated_at BEFORE UPDATE ON turfs
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Gyms
CREATE TABLE IF NOT EXISTS gyms (
  id             BIGSERIAL PRIMARY KEY,
  name           TEXT NOT NULL,
  description    TEXT NOT NULL DEFAULT '',
  address        TEXT NOT NULL DEFAULT '',
  monthly_price  NUMERIC NOT NULL DEFAULT 0,
  rating         NUMERIC NOT NULL DEFAULT 0,
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at     TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_gyms_updated_at ON gyms(updated_at);
DROP TRIGGER IF EXISTS trg_gyms_updated_at ON gyms;
CREATE TRIGGER trg_gyms_updated_at BEFORE UPDATE ON gyms
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Coaches
CREATE TABLE IF NOT EXISTS coaches (
  id                BIGSERIAL PRIMARY KEY,
  name              TEXT NOT NULL,
  bio               TEXT NOT NULL DEFAULT '',
  specialization    TEXT NOT NULL DEFAULT '',
  hourly_rate       NUMERIC NOT NULL DEFAULT 0,
  experience_years  INT NOT NULL DEFAULT 0,
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at        TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_coaches_updated_at ON coaches(updated_at);
DROP TRIGGER IF EXISTS trg_coaches_updated_at ON coaches;
CREATE TRIGGER trg_coaches_updated_at BEFORE UPDATE ON coaches
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Classes
CREATE TABLE IF NOT EXISTS classes (
  id            BIGSERIAL PRIMARY KEY,
  name          TEXT NOT NULL,
  description   TEXT NOT NULL DEFAULT '',
  duration_min  INT NOT NULL DEFAULT 0,
  level         TEXT NOT NULL DEFAULT '',
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at    TIMESTAMPTZ NULL
);
CREATE INDEX IF NOT EXISTS idx_classes_updated_at ON classes(updated_at);
DROP TRIGGER IF EXISTS trg_classes_updated_at ON classes;
CREATE TRIGGER trg_classes_updated_at BEFORE UPDATE ON classes
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Seed data: varied enough that crossfit/yoga/boxing each hit multiple categories.
INSERT INTO turfs (name, description, address, price_per_hour) VALUES
  ('Green Field Turf',     'Premium 5-a-side football turf with floodlights', 'MG Road',     1200),
  ('Cricket Pro Turf',     'Astroturf cricket pitch with bowling machines',  'HSR Layout',   900),
  ('Box Cricket Arena',    'Indoor box cricket facility, open all night',     'Whitefield',  1500),
  ('Sunrise Yoga Lawn',    'Outdoor lawn for sunrise yoga and fitness boot camps', 'Cubbon Park', 600);

INSERT INTO gyms (name, description, address, monthly_price, rating) VALUES
  ('CrossFit Downtown',     'Premier CrossFit box with certified L2 coaches', 'Indiranagar',   4500, 4.7),
  ('Iron Paradise',         'Classic bodybuilding gym with heavy free weights', 'Koramangala', 2500, 4.3),
  ('Yoga & Wellness Studio','Hatha, Vinyasa and meditation classes daily',     'Jayanagar',    3000, 4.8),
  ('Boxing Club 24x7',      'Boxing and kickboxing classes with sparring sessions', 'Marathahalli', 3500, 4.5);

INSERT INTO coaches (name, bio, specialization, hourly_rate, experience_years) VALUES
  ('Arjun Mehta', 'Certified CrossFit L2 coach with 8 years of experience',       'CrossFit', 1500,  8),
  ('Priya Nair',  'Yoga and mobility specialist, RYT-500 certified',              'Yoga',     1200, 10),
  ('Rohan Singh', 'Former national-level boxer, now coaching beginners and pros', 'Boxing',   1800, 12),
  ('Neha Verma',  'Sports nutritionist and strength coach for endurance athletes','Nutrition',2000,  6);

INSERT INTO classes (name, description, duration_min, level) VALUES
  ('Sunrise Yoga',          '60-minute Hatha yoga class for all levels',            60, 'Beginner'),
  ('CrossFit WOD',          'High-intensity workout of the day at CrossFit Downtown', 60, 'Intermediate'),
  ('Boxing Fundamentals',   'Stance, footwork and combinations for new boxers',     45, 'Beginner'),
  ('Spin Cycle',            'Indoor cycling with music-driven intervals',           45, 'All levels'),
  ('HIIT Express',          '30-minute high-intensity interval training',           30, 'Intermediate');
