use axum::http::header::{AUTHORIZATION, COOKIE, SET_COOKIE};
use axum::http::{HeaderMap, HeaderValue};
use axum::response::AppendHeaders;

use crate::config::Settings;
use crate::domain::iam::{AuthCredential, RequestContext, SessionTokens};

pub fn request_context_from_headers(headers: &HeaderMap, settings: &Settings) -> RequestContext {
    let product_code = header_value(headers, &settings.auth.context.product_header)
        .unwrap_or_else(|| settings.auth.context.default_product_code.clone());
    let client_type = header_value(headers, &settings.auth.context.client_type_header)
        .unwrap_or_else(|| settings.auth.context.default_client_type.clone());
    RequestContext {
        product_code,
        client_type,
    }
}

pub fn locale_from_headers(headers: &HeaderMap, settings: &Settings) -> String {
    header_value(headers, "X-Locale").unwrap_or_else(|| settings.i18n.default_locale.clone())
}

pub fn session_cookie(headers: &HeaderMap, settings: &Settings) -> Option<String> {
    cookie_value(headers, &settings.auth.cookie.name)
}

pub fn refresh_cookie(headers: &HeaderMap, settings: &Settings) -> Option<String> {
    cookie_value(headers, &settings.auth.refresh_cookie.name)
}

pub fn cookie_value(headers: &HeaderMap, cookie_name: &str) -> Option<String> {
    let cookie_header = headers.get(COOKIE)?.to_str().ok()?;
    cookie_header
        .split(';')
        .filter_map(|part| part.trim().split_once('='))
        .find_map(|(name, value)| (name == cookie_name).then(|| value.trim().to_string()))
}

pub fn bearer_token(headers: &HeaderMap) -> Option<String> {
    let value = headers.get(AUTHORIZATION)?.to_str().ok()?.trim();
    value
        .strip_prefix("Bearer ")
        .or_else(|| value.strip_prefix("bearer "))
        .map(str::trim)
        .filter(|token| !token.is_empty())
        .map(ToOwned::to_owned)
}

pub fn auth_credential_from_headers(headers: &HeaderMap, settings: &Settings) -> AuthCredential {
    AuthCredential {
        raw_session: session_cookie(headers, settings),
        raw_api_token: bearer_token(headers),
    }
}

pub fn set_auth_cookies(
    settings: &Settings,
    tokens: &SessionTokens,
) -> AppendHeaders<[(axum::http::HeaderName, HeaderValue); 2]> {
    AppendHeaders([
        (
            SET_COOKIE,
            cookie_header_value(
                &settings.auth.cookie.name,
                &tokens.raw_session,
                &settings.auth.cookie.path,
                settings.auth.session_ttl_seconds,
                &settings.auth.cookie.same_site,
                settings.auth.cookie.secure,
            ),
        ),
        (
            SET_COOKIE,
            cookie_header_value(
                &settings.auth.refresh_cookie.name,
                &tokens.raw_refresh,
                &settings.auth.refresh_cookie.path,
                settings.auth.refresh_ttl_seconds,
                &settings.auth.refresh_cookie.same_site,
                settings.auth.refresh_cookie.secure,
            ),
        ),
    ])
}

pub fn set_csrf_cookie(
    settings: &Settings,
) -> Option<AppendHeaders<[(axum::http::HeaderName, HeaderValue); 1]>> {
    settings.auth.csrf.enabled.then(|| {
        AppendHeaders([(
            SET_COOKIE,
            csrf_cookie_header_value(
                &settings.auth.csrf.cookie_name,
                &crypto::new_secret("csrf"),
                &settings.auth.csrf.path,
                settings.auth.csrf.ttl_seconds,
                &settings.auth.csrf.same_site,
                settings.auth.csrf.secure,
            ),
        )])
    })
}

pub fn clear_auth_cookies(
    settings: &Settings,
) -> AppendHeaders<[(axum::http::HeaderName, HeaderValue); 2]> {
    AppendHeaders([
        (
            SET_COOKIE,
            clear_cookie_header_value(
                &settings.auth.cookie.name,
                &settings.auth.cookie.path,
                &settings.auth.cookie.same_site,
                settings.auth.cookie.secure,
            ),
        ),
        (
            SET_COOKIE,
            clear_cookie_header_value(
                &settings.auth.refresh_cookie.name,
                &settings.auth.refresh_cookie.path,
                &settings.auth.refresh_cookie.same_site,
                settings.auth.refresh_cookie.secure,
            ),
        ),
    ])
}

fn cookie_header_value(
    name: &str,
    value: &str,
    path: &str,
    max_age: i64,
    same_site: &str,
    secure: bool,
) -> HeaderValue {
    let mut cookie =
        format!("{name}={value}; Path={path}; Max-Age={max_age}; SameSite={same_site}; HttpOnly");
    if secure {
        cookie.push_str("; Secure");
    }
    HeaderValue::from_str(&cookie).expect("valid Set-Cookie header")
}

fn clear_cookie_header_value(name: &str, path: &str, same_site: &str, secure: bool) -> HeaderValue {
    let mut cookie = format!("{name}=; Path={path}; Max-Age=0; SameSite={same_site}; HttpOnly");
    if secure {
        cookie.push_str("; Secure");
    }
    HeaderValue::from_str(&cookie).expect("valid Set-Cookie header")
}

fn csrf_cookie_header_value(
    name: &str,
    value: &str,
    path: &str,
    max_age: i64,
    same_site: &str,
    secure: bool,
) -> HeaderValue {
    let mut cookie =
        format!("{name}={value}; Path={path}; Max-Age={max_age}; SameSite={same_site}");
    if secure {
        cookie.push_str("; Secure");
    }
    HeaderValue::from_str(&cookie).expect("valid Set-Cookie header")
}

fn header_value(headers: &HeaderMap, name: &str) -> Option<String> {
    headers
        .get(name)
        .and_then(|value| value.to_str().ok())
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(ToOwned::to_owned)
}
