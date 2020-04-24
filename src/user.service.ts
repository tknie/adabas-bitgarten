import { config } from './config';
import { authHeader, authInitHeader } from './auth-header';
import store from './store';

export const userService = {
    login,
    logout,
    getAll,
    getAlbum,
};

function login(username: string , password: string) {
    const requestOptions = {
        method: 'POST',
        headers: authInitHeader(username, password),
    };

    return fetch(`${config.Url()}/login`, requestOptions)
        .then(handleResponse)
        .then((user) => {
            // login successful if there's a user in the response
            if (user) {
                // store user details and basic auth credentials in local storage
                // to keep user logged in between page refreshes
                user.authdata = window.btoa(username + ':' + password);
                user.username = username;
                localStorage.setItem('user', JSON.stringify(user));
            }

            return user;
        });
}

function logout() {
    // remove user from local storage to log user out
    localStorage.removeItem('user');
}

function getAll() {
    const requestOptions = {
        method: 'GET',
        headers: authHeader('application/json'),
    };

    return fetch(`${config.Url()}/rest/map/Album`, requestOptions).then(handleResponse);
}

function getAlbum(nr: number) {
    const requestOptions = {
        method: 'GET',
        headers: authHeader('application/json'),
    };
    return fetch(`${config.Url()}/rest/map/Album/${nr}`, requestOptions).then(handleResponse);
}

function handleResponse(response: any) {
    return response.text().then((text: any) => {
        const data = text && JSON.parse(text);
        if (!response.ok) {
            if (response.status === 401 || response.status === 404) {
                // auto logout if 401 response returned from api
                logout();
                location.reload(true);
            }

            const error = (data && data.message) || response.statusText;
            return Promise.reject(error);
        }
        return data;
    });
}

