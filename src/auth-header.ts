import axios from 'axios';

export function authHeader(content: string) {
    let acceptContent = 'application/json';
    if (content === '') {
        acceptContent = '';
    }
    // const response = { Authorization: '', Accept: acceptContent,
    // Origin: 'http://bitgarten.de' };
    const response = { Authorization: '', Accept: acceptContent };
    // return authorization header with basic auth credentials
    const x = localStorage.getItem('user');
    if (x === null) {
        return response;
        // return {Authorization: '', Accept: acceptContent};
    }
    const user = JSON.parse(x);
    if (user.token) {
        const bearerToken = 'Bearer ' + user.token;
        axios.defaults.headers.common.Authorization = bearerToken;
        axios.defaults.headers.common.Accept = content;
        response.Authorization = bearerToken;
        return response;
        // return { Authorization: bearerToken, Accept: acceptContent };
    }
    if (user && user.authdata) {
        response.Authorization = 'Basic ' + user.authdata;
        return response;
        // return { Authorization: 'Basic ' + user.authdata, Accept: acceptContent};
    }
    return response;
    // return {Authorization: '', Accept: acceptContent};
}

export function authInitHeader(username: string , password: string) {
    const b = btoa(username + ':' + password);
    return { Authorization: 'Basic ' + b, Accept: 'application/json' };
}

export function jwtAuth() {
    const x = localStorage.getItem('user');
    if (x === null) {
        return '';
    }
    const user = JSON.parse(x);
    return 'Bearer ' + user.token;
}

