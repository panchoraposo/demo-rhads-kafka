-- Local CDC source DB (mirrors software-templates/debezium-cdc-postgres helm init).
CREATE TABLE IF NOT EXISTS public.orders (
  id       SERIAL PRIMARY KEY,
  customer TEXT NOT NULL,
  amount   NUMERIC(12, 2) NOT NULL,
  status   TEXT NOT NULL DEFAULT 'new'
);

ALTER ROLE orders WITH REPLICATION;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_publication WHERE pubname = 'dbz_publication') THEN
    CREATE PUBLICATION dbz_publication FOR TABLE public.orders;
  END IF;
END
$$;

INSERT INTO public.orders (customer, amount, status)
VALUES ('local-seed', 1.00, 'new')
ON CONFLICT DO NOTHING;
