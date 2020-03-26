import {Button, Nav, Navbar} from "react-bootstrap";
import React from "react";
import NavItem from "react-bootstrap/NavItem";

/**
 * Function to draw the page header
 * @param props the property to check whether it is logged or not, to draw the apposite component
 * @return the component to be drawn
 */
export default function Header(props) {
    const toDraw = props.logged? <Button variant="outline-light" onClick={props.logout}>Logout</Button> : <img src={require('../assets/logo_poli3.png')} height="50px" alt=""/>;
    const name = props.adminHidden ? "Professor Area" : "Student Area";
    const adminBtn = props.renderAdminBtn ? <Button variant="outline-light" onClick={props.switchAdminView}>{name}</Button> : <div/>;
    return <header>
        <Navbar bg="dark" variant="dark" expand="lg">
            <Navbar.Brand href="">CrownLabs</Navbar.Brand>
            <Nav className="ml-auto" as="ul">
                <NavItem as="li" className="mr-2">
                    {adminBtn}
                </NavItem>
                <NavItem as="li">
                    {toDraw}
                </NavItem>
            </Nav>
        </Navbar>
    </header>
}