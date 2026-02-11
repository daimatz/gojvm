public class AbstractClass {
    static abstract class Shape {
        String name;
        Shape(String name) { this.name = name; }
        abstract double area();
        String describe() { return name + ": " + area(); }
    }

    static class Circle extends Shape {
        double radius;
        Circle(double r) { super("Circle"); this.radius = r; }
        double area() { return 3.14159 * radius * radius; }
    }

    static class Rect extends Shape {
        double w, h;
        Rect(double w, double h) { super("Rect"); this.w = w; this.h = h; }
        double area() { return w * h; }
    }

    public static void main(String[] args) {
        Shape[] shapes = { new Circle(5), new Rect(3, 4) };
        for (Shape s : shapes) {
            System.out.println(s.describe());
        }
        System.out.println(shapes[0] instanceof Shape); // true
        System.out.println(shapes[0] instanceof Circle); // true
    }
}
