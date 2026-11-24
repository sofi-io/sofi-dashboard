--
-- PostgreSQL database dump
--

\restrict lqAGHtBzPgmjvefCbUK1Wnmu40J5vQa5dEqTXvOGmwakRLtjLpR3KtgTEwOaxUN

-- Dumped from database version 16.16
-- Dumped by pg_dump version 16.16

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- Name: EXTENSION pgcrypto; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: club_sources; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.club_sources (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    club_id uuid NOT NULL,
    source_key text NOT NULL,
    external_id text NOT NULL,
    source_url text,
    discovered_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: clubs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.clubs (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    canonical_name text NOT NULL,
    slug text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: competition_sources; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.competition_sources (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    competition_id uuid NOT NULL,
    source_key text NOT NULL,
    external_id text NOT NULL,
    source_url text NOT NULL,
    discovered_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: competitions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.competitions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    organization_id uuid,
    name text NOT NULL,
    slug text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    canonical_key text NOT NULL
);


--
-- Name: documents; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.documents (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    match_id uuid,
    source_key text NOT NULL,
    source_url text,
    original_filename text,
    content_type text,
    size_bytes bigint,
    sha256 character(64),
    storage_key text,
    status text DEFAULT 'discovered'::text NOT NULL,
    discovered_at timestamp with time zone DEFAULT now() NOT NULL,
    downloaded_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    document_type text DEFAULT 'unknown'::text NOT NULL,
    download_attempts integer DEFAULT 0 NOT NULL,
    download_started_at timestamp with time zone,
    last_error text,
    download_token uuid
);


--
-- Name: extractions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.extractions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    document_id uuid NOT NULL,
    provider text DEFAULT 'sofi-api'::text NOT NULL,
    external_extraction_id text,
    status text DEFAULT 'pending'::text NOT NULL,
    raw_result jsonb,
    error_message text,
    submitted_at timestamp with time zone,
    completed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    submission_attempts integer DEFAULT 0 NOT NULL,
    submission_started_at timestamp with time zone,
    submission_token uuid,
    next_poll_at timestamp with time zone,
    poll_started_at timestamp with time zone,
    poll_token uuid,
    last_polled_at timestamp with time zone,
    CONSTRAINT extractions_submission_attempts_check CHECK ((submission_attempts >= 0))
);


--
-- Name: match_sources; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.match_sources (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    match_id uuid NOT NULL,
    source_key text NOT NULL,
    external_id text,
    source_url text,
    source_payload jsonb,
    discovered_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: matches; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.matches (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    season_id uuid NOT NULL,
    home_club_id uuid NOT NULL,
    away_club_id uuid NOT NULL,
    kickoff_at timestamp with time zone,
    round_name text,
    venue text,
    status text DEFAULT 'scheduled'::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    phase_id uuid,
    round_number integer,
    source_date_text text,
    source_time_text text,
    CONSTRAINT matches_check CHECK ((home_club_id <> away_club_id))
);


--
-- Name: organizations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.organizations (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    slug text NOT NULL,
    website_url text,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: phase_sources; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.phase_sources (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    phase_id uuid NOT NULL,
    source_key text NOT NULL,
    external_id text NOT NULL,
    source_url text,
    discovered_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: phases; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.phases (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    season_id uuid NOT NULL,
    name text NOT NULL,
    phase_order integer,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);


--
-- Name: season_sources; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.season_sources (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    season_id uuid NOT NULL,
    source_key text NOT NULL,
    external_id text NOT NULL,
    source_url text,
    discovered_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: seasons; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.seasons (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    competition_id uuid NOT NULL,
    name text NOT NULL,
    starts_on date,
    ends_on date,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: club_sources club_sources_club_id_source_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.club_sources
    ADD CONSTRAINT club_sources_club_id_source_key_key UNIQUE (club_id, source_key);


--
-- Name: club_sources club_sources_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.club_sources
    ADD CONSTRAINT club_sources_pkey PRIMARY KEY (id);


--
-- Name: club_sources club_sources_source_key_external_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.club_sources
    ADD CONSTRAINT club_sources_source_key_external_id_key UNIQUE (source_key, external_id);


--
-- Name: clubs clubs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.clubs
    ADD CONSTRAINT clubs_pkey PRIMARY KEY (id);


--
-- Name: clubs clubs_slug_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.clubs
    ADD CONSTRAINT clubs_slug_key UNIQUE (slug);


--
-- Name: competition_sources competition_sources_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.competition_sources
    ADD CONSTRAINT competition_sources_pkey PRIMARY KEY (id);


--
-- Name: competition_sources competition_sources_source_key_external_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.competition_sources
    ADD CONSTRAINT competition_sources_source_key_external_id_key UNIQUE (source_key, external_id);


--
-- Name: competitions competitions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.competitions
    ADD CONSTRAINT competitions_pkey PRIMARY KEY (id);


--
-- Name: competitions competitions_slug_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.competitions
    ADD CONSTRAINT competitions_slug_key UNIQUE (slug);


--
-- Name: documents documents_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.documents
    ADD CONSTRAINT documents_pkey PRIMARY KEY (id);


--
-- Name: documents documents_source_key_source_url_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.documents
    ADD CONSTRAINT documents_source_key_source_url_key UNIQUE (source_key, source_url);


--
-- Name: extractions extractions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.extractions
    ADD CONSTRAINT extractions_pkey PRIMARY KEY (id);


--
-- Name: extractions extractions_provider_external_extraction_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.extractions
    ADD CONSTRAINT extractions_provider_external_extraction_id_key UNIQUE (provider, external_extraction_id);


--
-- Name: match_sources match_sources_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.match_sources
    ADD CONSTRAINT match_sources_pkey PRIMARY KEY (id);


--
-- Name: match_sources match_sources_source_key_external_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.match_sources
    ADD CONSTRAINT match_sources_source_key_external_id_key UNIQUE (source_key, external_id);


--
-- Name: matches matches_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.matches
    ADD CONSTRAINT matches_pkey PRIMARY KEY (id);


--
-- Name: organizations organizations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organizations
    ADD CONSTRAINT organizations_pkey PRIMARY KEY (id);


--
-- Name: organizations organizations_slug_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.organizations
    ADD CONSTRAINT organizations_slug_key UNIQUE (slug);


--
-- Name: phase_sources phase_sources_phase_id_source_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.phase_sources
    ADD CONSTRAINT phase_sources_phase_id_source_key_key UNIQUE (phase_id, source_key);


--
-- Name: phase_sources phase_sources_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.phase_sources
    ADD CONSTRAINT phase_sources_pkey PRIMARY KEY (id);


--
-- Name: phase_sources phase_sources_source_key_external_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.phase_sources
    ADD CONSTRAINT phase_sources_source_key_external_id_key UNIQUE (source_key, external_id);


--
-- Name: phases phases_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.phases
    ADD CONSTRAINT phases_pkey PRIMARY KEY (id);


--
-- Name: phases phases_season_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.phases
    ADD CONSTRAINT phases_season_id_name_key UNIQUE (season_id, name);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: season_sources season_sources_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.season_sources
    ADD CONSTRAINT season_sources_pkey PRIMARY KEY (id);


--
-- Name: season_sources season_sources_season_id_source_key_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.season_sources
    ADD CONSTRAINT season_sources_season_id_source_key_key UNIQUE (season_id, source_key);


--
-- Name: season_sources season_sources_source_key_external_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.season_sources
    ADD CONSTRAINT season_sources_source_key_external_id_key UNIQUE (source_key, external_id);


--
-- Name: seasons seasons_competition_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.seasons
    ADD CONSTRAINT seasons_competition_id_name_key UNIQUE (competition_id, name);


--
-- Name: seasons seasons_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.seasons
    ADD CONSTRAINT seasons_pkey PRIMARY KEY (id);


--
-- Name: competitions_canonical_key_uidx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX competitions_canonical_key_uidx ON public.competitions USING btree (canonical_key);


--
-- Name: documents_download_queue_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX documents_download_queue_idx ON public.documents USING btree (discovered_at) WHERE ((status = ANY (ARRAY['discovered'::text, 'failed'::text])) AND (source_url IS NOT NULL));


--
-- Name: documents_download_stale_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX documents_download_stale_idx ON public.documents USING btree (download_started_at) WHERE (status = 'downloading'::text);


--
-- Name: documents_match_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX documents_match_idx ON public.documents USING btree (match_id);


--
-- Name: documents_pending_type_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX documents_pending_type_idx ON public.documents USING btree (match_id, source_key, document_type) WHERE (source_url IS NULL);


--
-- Name: documents_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX documents_status_idx ON public.documents USING btree (status);


--
-- Name: extractions_active_document_provider_uidx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX extractions_active_document_provider_uidx ON public.extractions USING btree (document_id, provider) WHERE (status = ANY (ARRAY['pending'::text, 'submitting'::text, 'submitted'::text, 'processing'::text, 'submission_unknown'::text]));


--
-- Name: extractions_document_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX extractions_document_idx ON public.extractions USING btree (document_id);


--
-- Name: extractions_status_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX extractions_status_idx ON public.extractions USING btree (status);


--
-- Name: matches_phase_round_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX matches_phase_round_idx ON public.matches USING btree (phase_id, round_number);


--
-- Name: matches_season_kickoff_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX matches_season_kickoff_idx ON public.matches USING btree (season_id, kickoff_at);


--
-- Name: club_sources club_sources_club_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.club_sources
    ADD CONSTRAINT club_sources_club_id_fkey FOREIGN KEY (club_id) REFERENCES public.clubs(id) ON DELETE CASCADE;


--
-- Name: competition_sources competition_sources_competition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.competition_sources
    ADD CONSTRAINT competition_sources_competition_id_fkey FOREIGN KEY (competition_id) REFERENCES public.competitions(id) ON DELETE CASCADE;


--
-- Name: competitions competitions_organization_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.competitions
    ADD CONSTRAINT competitions_organization_id_fkey FOREIGN KEY (organization_id) REFERENCES public.organizations(id);


--
-- Name: documents documents_match_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.documents
    ADD CONSTRAINT documents_match_id_fkey FOREIGN KEY (match_id) REFERENCES public.matches(id) ON DELETE SET NULL;


--
-- Name: extractions extractions_document_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.extractions
    ADD CONSTRAINT extractions_document_id_fkey FOREIGN KEY (document_id) REFERENCES public.documents(id) ON DELETE RESTRICT;


--
-- Name: match_sources match_sources_match_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.match_sources
    ADD CONSTRAINT match_sources_match_id_fkey FOREIGN KEY (match_id) REFERENCES public.matches(id) ON DELETE CASCADE;


--
-- Name: matches matches_away_club_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.matches
    ADD CONSTRAINT matches_away_club_id_fkey FOREIGN KEY (away_club_id) REFERENCES public.clubs(id);


--
-- Name: matches matches_home_club_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.matches
    ADD CONSTRAINT matches_home_club_id_fkey FOREIGN KEY (home_club_id) REFERENCES public.clubs(id);


--
-- Name: matches matches_phase_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.matches
    ADD CONSTRAINT matches_phase_id_fkey FOREIGN KEY (phase_id) REFERENCES public.phases(id);


--
-- Name: matches matches_season_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.matches
    ADD CONSTRAINT matches_season_id_fkey FOREIGN KEY (season_id) REFERENCES public.seasons(id) ON DELETE RESTRICT;


--
-- Name: phase_sources phase_sources_phase_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.phase_sources
    ADD CONSTRAINT phase_sources_phase_id_fkey FOREIGN KEY (phase_id) REFERENCES public.phases(id) ON DELETE CASCADE;


--
-- Name: phases phases_season_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.phases
    ADD CONSTRAINT phases_season_id_fkey FOREIGN KEY (season_id) REFERENCES public.seasons(id) ON DELETE CASCADE;


--
-- Name: season_sources season_sources_season_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.season_sources
    ADD CONSTRAINT season_sources_season_id_fkey FOREIGN KEY (season_id) REFERENCES public.seasons(id) ON DELETE CASCADE;


--
-- Name: seasons seasons_competition_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.seasons
    ADD CONSTRAINT seasons_competition_id_fkey FOREIGN KEY (competition_id) REFERENCES public.competitions(id) ON DELETE RESTRICT;


--
-- PostgreSQL database dump complete
--

\unrestrict lqAGHtBzPgmjvefCbUK1Wnmu40J5vQa5dEqTXvOGmwakRLtjLpR3KtgTEwOaxUN

